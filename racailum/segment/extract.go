package segment

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/limiter"
	"github.com/dtynn/londobell/racailum/segment/actor"
	"github.com/dtynn/londobell/racailum/segment/extract"
	east "github.com/dtynn/londobell/racailum/segment/extract/actorstate"
	ets "github.com/dtynn/londobell/racailum/segment/extract/tipset"
)

type persistCtx struct {
	ctx                   context.Context
	actorSet              *actor.Set
	log                   *zap.SugaredLogger
	asyncPersistWaitGroup multierror.Group
}

func (s *Segment) extractTipSets(ctx context.Context, tss []*common.LinkedTipSet) error {
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
	aset, err := actor.NewSet(ctx, s.dal.StateManager, tss[size-1])
	if err != nil {
		return err
	}

	pctx := &persistCtx{
		ctx:      ctx,
		actorSet: aset,
		log:      elog,
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
		if err := s.extractPart(pctx, part); err != nil {
			return err
		}

		tsDone += len(part)
		partDone++
		elog.Infow("part done", "done-parts", partDone, "done-tss", tsDone)
	}

	if err := pctx.asyncPersistWaitGroup.Wait(); err != nil {
		elog.Errorf("error occurs in async persist: %s", err)
	}

	elog.Info("all parts done")
	return nil
}

func (s *Segment) extractPart(ctx *persistCtx, part []*common.LinkedTipSet) error {
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

	ectx, err := extract.NewCtx(innerCtx, s.dal, elog, ctx.actorSet, s.opts.Extract.ExtractOptions)
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

			res := extract.NewRes(1024)

			err = ets.Extract(ectx, res, ts)
			if err != nil {
				return common.NonCtxCanceledErr(err)
			}

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

	// reset context
	ectx.C = ctx.ctx
	for rhi := range regulars {
		if rheads := regulars[rhi]; len(rheads) > 0 {
			if err := s.extractRegularStates(ectx, rheads); err != nil {
				return fmt.Errorf("#%d regular heads: %w", rhi, err)
			}
		}
	}

	return s.extractDiffStates(ectx)
}

func (s *Segment) extractRegularStates(ctx *extract.Ctx, heads []*common.ActorHead) error {
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

			res := extract.NewRes(1024)

			err = east.ExtractRegular(ctx, res, head)
			if err != nil {
				return common.NonCtxCanceledErr(err)
			}

			docs[hi] = res.Docs

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("extract part regular states: %w", err)
	}

	if err := s.insertMany(originCtx, ctx.L, docs); err != nil {
		return fmt.Errorf("insert extracted documents from regular states: %w", err)
	}

	return nil
}

func (s *Segment) extractDiffStates(ctx *extract.Ctx) error {
	ctx.Actors.Head.Lock()
	heads := ctx.Actors.Head.M
	ctx.Actors.Head.Unlock()

	if len(heads) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		ctx.L.Infow("actor diff states extracting done", "elapsed", time.Now().Sub(start).String())
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

	i := 0
	for key := range heads {
		head := heads[key]
		hi := i
		i++

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

			res := extract.NewRes(16)

			err = east.ExtractDiff(ctx, res, head)
			if err != nil {
				return common.NonCtxCanceledErr(err)
			}

			docs[hi] = res.Docs

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("extract part diff states: %w", err)
	}

	if err := s.insertMany(originCtx, ctx.L, docs); err != nil {
		return fmt.Errorf("insert extracted documents from diff states: %w", err)
	}

	return nil
}

// DryExtract tries to extract all results from give tipset
func (s *Segment) DryExtract(ctx context.Context, ts *common.LinkedTipSet) ([]*extract.Res, error) {
	dryOptions := extract.DryOptions()
	aset, err := actor.NewSet(ctx, s.dal.StateManager, ts)
	if err != nil {
		return nil, fmt.Errorf("new actor set: %w", err)
	}

	dlog := log.With("dry", true)
	ectx, err := extract.NewCtx(ctx, s.dal, dlog, aset, dryOptions)
	if err != nil {
		return nil, fmt.Errorf("new extract context: %w", err)
	}

	tres := extract.NewRes(1024)

	err = ets.Extract(ectx, tres, ts)
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

			res := extract.NewRes(1024)

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
