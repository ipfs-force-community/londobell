package segment

import (
	"context"
	"fmt"
	"time"

	"go.opencensus.io/stats"

	"github.com/ipfs-force-community/londobell/metrics"

	"go.opencensus.io/trace"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/limiter"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	east "github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/tipset"
	ets "github.com/ipfs-force-community/londobell/racailum/segment/extract/tipset"
)

type persistCtx struct {
	ctx                   context.Context
	actorSet              *actor.Set
	log                   *zap.SugaredLogger
	asyncPersistWaitGroup multierror.Group
	latestDealID          int64
}

func (s *Segment) ExtractTipSets(ctx context.Context, tss []*common.LinkedTipSet, tmp bool) error {
	ctx, span := trace.StartSpan(ctx, "segment.ExtractTipSets")
	defer span.End()

	if len(tss) == 0 {
		return nil
	}

	elog := log.With("range", common.FormatTipSetEpochRange(tss))

	elog.Info("tipsets extracting started")
	start := time.Now()
	defer func() {
		elog.Infow("tipsets extracting done", "elapsed", time.Now().Sub(start).String())
	}()

	size := len(tss)
	var (
		aset         *actor.Set
		latestDealID int64
		err          error
	)
	if !tmp {
		aset, err = actor.NewSet(ctx, s.dal.StateManager, tss[size-1], tmp)
		if err != nil {
			return err
		}

		latestDealID, err = s.GetLatestDealID(ctx)
		if err != nil {
			return err
		}
	} else {
		ts := tss[0]
		version := s.dal.GetNetworkVersion(ctx, ts.Height())
		if Actorset == nil {
			aset, err = actor.NewSet(ctx, s.dal.StateManager, ts, tmp)
			if err != nil {
				return err
			}
			Actorset = &ActorSet{
				Version: version,
				Set:     aset,
			}
		} else {
			if Actorset.Version != version {
				if s.opts.Extract.ExtractOptions.SkipExpensiveEpoch && tipset.IsExpensive(ctx, s.dal.StateManager, ts) {
					// TODO: extract simple invoc results here
					elog.Warn("ignore expensive epoch actor load")
				} else {
					aset, err = actor.NewSet(ctx, s.dal.StateManager, tss[0], tmp)
					if err != nil {
						return err
					}
					Actorset = &ActorSet{
						Version: version,
						Set:     aset,
					}
					elog.Infof("reload new version: %s actor", version)
				}
			} else {
				aset = Actorset.Set
			}

		}

	}

	pctx := &persistCtx{
		ctx:          ctx,
		actorSet:     aset,
		log:          elog,
		latestDealID: latestDealID,
	}

	tsDone := 0
	partDone := 0
	for tsDone < size {
		start := tsDone
		end := start + s.opts.Extract.TipSetPartSizeLimit
		if end > size {
			end = size
		}

		part := tss[start:end]
		if err := s.extractPart(pctx, part, tmp); err != nil {
			return err
		}

		tsDone += len(part)
		partDone++
		elog.Infow("part done", "done-parts", partDone, "done-tss", tsDone)
	}

	if err := pctx.asyncPersistWaitGroup.Wait(); err != nil {
		return fmt.Errorf("error occurs in async persist: %s", err)
	}

	elog.Info("all parts done")
	return nil
}

func (s *Segment) extractPart(ctx *persistCtx, part []*common.LinkedTipSet, tmp bool) error {
	if len(part) == 0 {
		return nil
	}

	elog := ctx.log.With("part", common.FormatTipSetEpochRange(part))
	start := time.Now()
	defer func() {
		elog.Infow("tipset part extracting done", "elapsed", time.Now().Sub(start).String())
	}()

	innerCtx, innerCancel := context.WithCancel(ctx.ctx)
	defer innerCancel()

	ectx, err := extract.NewCtx(innerCtx, s.dal, elog, ctx.actorSet, ctx.latestDealID, s.opts.Extract.ExtractOptions)
	if err != nil {
		return err
	}

	var ewg multierror.Group
	lim := limiter.New(s.opts.Extract.TipSetJobLimit)

	docs := make([][]common.Document, len(part))
	regulars := make([][]*common.ActorHead, len(part))
	for ti := range part {
		ti := ti
		ts := part[ti]
		ewg.Go(func() error {
			if !lim.Acquire(innerCtx) {
				return nil
			}

			defer func() {
				lim.Release(innerCtx)
			}()

			// fail fast
			select {
			case <-innerCtx.Done():
				return nil

			default:
			}

			var err error
			defer func() {
				if err != nil {
					innerCancel()
				}
			}()

			forRegular := ectx.Opts.StateRegular.Interval > 0 && ts.Height()%ectx.Opts.StateRegular.Interval == 0
			regCap := 0
			if forRegular {
				regCap = 700000
			}

			res := extract.NewRes(4096, regCap)

			err = ets.Extract(ectx, res, ts, tmp)
			if err != nil {
				return common.NonCtxCanceledErr(err)
			}

			log.Infof("after Extract res docs lens %d res regular states %d\n", len(res.Docs), len(res.RegularStates))

			docs[ti] = res.Docs
			regulars[ti] = res.RegularStates
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("extract part: %w", err)
	}

	if s.opts.Persist.Async {
		ctx.asyncPersistWaitGroup.Go(func() error {
			if err := s.insertMany(ctx.ctx, elog, docs); err != nil {
				if nerr := common.NonCtxCanceledErr(err); nerr != nil {
					stats.Record(ctx.ctx, metrics.ExtractError.M(1))
					elog.Errorf("insert extracted documents from tipsets: %s", err)
					return nerr
				}
			}

			return nil
		})
	} else {
		if err := s.insertMany(ctx.ctx, elog, docs); err != nil {
			return fmt.Errorf("insert extracted documents from tipsets: %w", err)
		}
	}

	// temporary db don't need to store state datas
	if tmp {
		return nil
	}

	// reset context
	ectx.C = ctx.ctx
	for rhi := range regulars {
		if rheads := regulars[rhi]; len(rheads) > 0 {
			if err := s.extractRegularStates(ectx, ctx, rheads); err != nil {
				return fmt.Errorf("#%d regular heads: %w", rhi, err)
			}
		}
	}

	return nil
}

func (s *Segment) extractRegularStates(ctx *extract.Ctx, pctx *persistCtx, heads []*common.ActorHead) error {
	if len(heads) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		ctx.L.Infow("actor regular states extracting done", "elapsed", time.Now().Sub(start).String())
	}()

	originCtx := ctx.C
	innerCtx, innerCancel := context.WithCancel(originCtx)
	defer innerCancel()

	ctx.C = innerCtx
	defer func() {
		ctx.C = originCtx
	}()

	var ewg multierror.Group
	docs := make([][]common.Document, len(heads))
	lim := limiter.New(s.opts.Extract.StateJobLimit)

	for hi := range heads {
		hi := hi

		head := heads[hi]

		ewg.Go(func() error {
			if !lim.Acquire(innerCtx) {
				return nil
			}

			defer func() {
				lim.Release(innerCtx)
			}()

			select {
			case <-innerCtx.Done():
				return nil

			default:
			}

			var err error
			defer func() {
				if err != nil {
					innerCancel()
				}
			}()

			res := extract.NewRes(8, 0)

			err = east.ExtractRegular(ctx, res, head)
			if err != nil {
				return common.NonCtxCanceledErr(err)
			}

			if len(res.Docs) > 8 {
				log.Infof("after ExtractRegular res len is %d reg len is %d\n", len(res.Docs), len(res.RegularStates))
			}

			docs[hi] = res.Docs

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("extract part regular states: %w", err)
	}

	if s.opts.Persist.AsyncState {
		pctx.asyncPersistWaitGroup.Go(func() error {
			if err := s.insertMany(originCtx, ctx.L, docs); err != nil {
				stats.Record(originCtx, metrics.ExtractError.M(1))
				return fmt.Errorf("insert extracted documents from regular states: %w", err)
			}
			return nil
		})
	} else {
		if err := s.insertMany(originCtx, ctx.L, docs); err != nil {
			return fmt.Errorf("insert extracted documents from regular states: %w", err)
		}
	}

	return nil
}

// DryExtract tries to extract all results from give tipset
func (s *Segment) DryExtract(ctx context.Context, ts *common.LinkedTipSet, allowNilChild bool) ([]*extract.Res, error) {
	dryOptions := extract.DryOptions()
	aset, err := actor.NewSet(ctx, s.dal.StateManager, ts, allowNilChild)
	if err != nil {
		return nil, fmt.Errorf("new actor set: %w", err)
	}

	latestDealID := int64(-1)

	dlog := log.With("dry", true)
	ectx, err := extract.NewCtx(ctx, s.dal, dlog, aset, latestDealID, dryOptions)
	if err != nil {
		return nil, fmt.Errorf("new extract context: %w", err)
	}

	tres := extract.NewRes(1024, 0)

	err = ets.Extract(ectx, tres, ts, allowNilChild)
	if err != nil {
		return nil, fmt.Errorf("extract tipset results: %w", err)
	}

	results := make([]*extract.Res, len(tres.RegularStates)+1)
	results[0] = tres

	if len(tres.RegularStates) == 0 {
		return results, nil
	}

	innerCtx, innerCancel := context.WithCancel(ctx)
	defer innerCancel()

	var ewg multierror.Group

	actres := results[1:]
	lim := limiter.New(s.opts.Extract.StateJobLimit)

	for hi := range tres.RegularStates {
		hi := hi

		head := tres.RegularStates[hi]

		ewg.Go(func() error {
			if !lim.Acquire(innerCtx) {
				return nil
			}

			defer func() {
				lim.Release(innerCtx)
			}()

			select {
			case <-innerCtx.Done():
				return nil

			default:
			}

			var err error
			defer func() {
				if err != nil {
					innerCancel()
				}
			}()

			res := extract.NewRes(1024, 0)

			err = east.ExtractRegular(ectx, res, head)
			if err != nil {
				return common.NonCtxCanceledErr(err)
			}

			actres[hi] = res

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, fmt.Errorf("extract part regular states: %w", err)
	}

	return results, nil
}
