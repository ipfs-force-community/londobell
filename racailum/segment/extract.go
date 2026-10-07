package segment

import (
	"context"
	"fmt"
	"time"

	"go.opencensus.io/stats"

	"github.com/ipfs-force-community/londobell/metrics"

	"go.opencensus.io/trace"

	"github.com/hashicorp/go-multierror"
	bf "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	v0api "github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"

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
	persistSem            chan struct{}
}

// rawChainIO provides ChainReadObj/ChainHasObj from the FullNode API,
// used to create an uncached blockstore for event AMT loading.
type rawChainIO struct {
	api v0api.FullNode
}

func (cio *rawChainIO) ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error) {
	return cio.api.ChainReadObj(ctx, c)
}

func (cio *rawChainIO) ChainHasObj(ctx context.Context, c cid.Cid) (bool, error) {
	return cio.api.ChainHasObj(ctx, c)
}

func (cio *rawChainIO) ChainPutObj(ctx context.Context, blk bf.Block) error {
	return nil
}

// jobRetryBackoff 是作业重试前的固定退避（变量形式便于测试缩短）。
var jobRetryBackoff = 3 * time.Second

// extractJobTimeoutErr 表示「作业在 deadline 内没有返回」这一类错误。
type extractJobTimeoutErr struct {
	what    string
	timeout time.Duration
}

func (e *extractJobTimeoutErr) Error() string {
	return fmt.Sprintf("%s: extract job timeout after %s", e.what, e.timeout)
}

// extractJobWithTimeout 以「单次尝试带 deadline + 失败重试」执行一个抽取作业。
//
// 为什么要在独立 goroutine 里跑并「超时即放弃」而不是只给 context 加 deadline：
// 挂死点位于 go-jsonrpc 客户端内部（等一个永不送达的响应 / nil channel），它不保证
// 响应 context 取消，所以必须允许放弃该次尝试。被放弃的 goroutine 会泄漏，但有上限
// （重试次数 × 并发作业数），且它只写自己那份 *extract.Res，不与成功的那次共享状态。
//
// 重试用尽后返回错误，上层（Segment.Extract → RaCailum.Run 循环）会在下一个 tipset
// 到来时按已提交的 final_height 整段重跑，因此这里不需要更复杂的恢复逻辑。
func (s *Segment) extractJobWithTimeout(
	parent context.Context,
	timeout time.Duration,
	attempts int,
	elog *zap.SugaredLogger,
	what string,
	fn func(ctx context.Context) (*extract.Res, error),
) (*extract.Res, error) {
	if attempts < 1 {
		attempts = 1
	}
	if timeout <= 0 {
		timeout = 20 * time.Minute
	}

	type jobResult struct {
		res *extract.Res
		err error
	}

	var lastErr error
	for i := 1; i <= attempts; i++ {
		if err := parent.Err(); err != nil {
			return nil, err
		}

		jobCtx, cancel := context.WithTimeout(parent, timeout)
		done := make(chan jobResult, 1) // 缓冲 1：即使本次尝试被放弃，其 goroutine 也不会阻塞
		go func() {
			res, err := fn(jobCtx)
			done <- jobResult{res: res, err: err}
		}()

		var (
			res     *extract.Res
			err     error
			expired bool
		)
		select {
		case r := <-done:
			res, err = r.res, r.err
		case <-jobCtx.Done():
			expired = true
			err = &extractJobTimeoutErr{what: what, timeout: timeout}
		}
		cancel()

		if err == nil {
			return res, nil
		}
		if parent.Err() != nil {
			return nil, parent.Err()
		}

		lastErr = err
		if expired {
			stats.Record(parent, metrics.ExtractError.M(1))
		}
		if i < attempts {
			elog.Warnw("extract job failed, will retry",
				"what", what, "attempt", i, "of", attempts,
				"timeout", timeout.String(), "expired", expired, "err", err)
			select {
			case <-time.After(jobRetryBackoff):
			case <-parent.Done():
				return nil, parent.Err()
			}
		}
	}

	return nil, fmt.Errorf("extract job failed after %d attempt(s): %w", attempts, lastErr)
}

// waitAsyncPersist 等待异步落库完成，超过 timeout 视为失败。
// 与作业超时同理：Mongo 写入挂住时不能永久等待（否则整批静默停摆）。
//
// 注意：multierror.Group.Wait() 返回具体类型 *multierror.Error，nil 值装进 error 接口后
// 不再等于 nil（typed-nil），所以这里用具体类型接收、显式判空后再返回。
func waitAsyncPersist(g *multierror.Group, timeout time.Duration) error {
	if timeout <= 0 {
		if e := g.Wait(); e != nil {
			return e
		}
		return nil
	}

	done := make(chan *multierror.Error, 1)
	go func() { done <- g.Wait() }()

	select {
	case e := <-done:
		if e == nil {
			return nil
		}
		return e
	case <-time.After(timeout):
		return fmt.Errorf("async persist wait timeout after %s", timeout)
	}
}

// insertManyWithTimeout 同步落库，带超时（Mongo driver 会响应 context 取消）。
func (s *Segment) insertManyWithTimeout(ctx context.Context, l *zap.SugaredLogger, docSets [][]common.Document) error {
	timeout := s.opts.Persist.WaitTimeout
	if timeout <= 0 {
		return s.insertMany(ctx, l, docSets)
	}

	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return s.insertMany(tctx, l, docSets)
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
		currentActorset := loadActorSet()
		if currentActorset == nil {
			aset, err = actor.NewSet(ctx, s.dal.StateManager, ts, tmp)
			if err != nil {
				return err
			}
			storeActorSet(&ActorSet{
				Version: version,
				Set:     aset,
			})
		} else {
			if currentActorset.Version != version {
				if s.opts.Extract.ExtractOptions.SkipExpensiveEpoch && tipset.IsExpensive(ctx, s.dal.StateManager, ts) {
					// TODO: extract simple invoc results here
					elog.Warn("ignore expensive epoch actor load")
				} else {
					aset, err = actor.NewSet(ctx, s.dal.StateManager, tss[0], tmp)
					if err != nil {
						return err
					}
					storeActorSet(&ActorSet{
						Version: version,
						Set:     aset,
					})
					elog.Infof("reload new version: %s actor", version)
				}
			} else {
				aset = currentActorset.Set
			}

		}

	}

	pctx := &persistCtx{
		ctx:          ctx,
		actorSet:     aset,
		log:          elog,
		latestDealID: latestDealID,
		persistSem:   make(chan struct{}, 2),
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

	if err := waitAsyncPersist(&pctx.asyncPersistWaitGroup, s.opts.Persist.WaitTimeout); err != nil {
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

	var eventsBlockstore blockstore.Blockstore
	if s.fullNode != nil {
		fullAPI := s.fullNode.GetAppropriateAPI()
		eventsBlockstore = blockstore.NewAPIBlockstore(&rawChainIO{api: fullAPI})
	}

	ectx, err := extract.NewCtx(innerCtx, s.dal, elog, ctx.actorSet, ctx.latestDealID, s.opts.Extract.ExtractOptions, s.fullNode, eventsBlockstore)
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

			// 每个 tipset 作业独立超时 + 重试：节点 RPC 卡死不再让整批永久停摆。
			res, err := s.extractJobWithTimeout(innerCtx, s.opts.Extract.TipSetJobTimeout, s.opts.Extract.JobRetry+1, elog,
				fmt.Sprintf("tipset %d", ts.Height()), func(jobCtx context.Context) (*extract.Res, error) {
					jctx := *ectx
					jctx.C = jobCtx
					jres := extract.NewRes(4096, regCap)
					if err := ets.Extract(&jctx, jres, ts, tmp); err != nil {
						return nil, err
					}
					return jres, nil
				})
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
		select {
		case ctx.persistSem <- struct{}{}:
		case <-ctx.ctx.Done():
			return ctx.ctx.Err()
		}

		ctx.asyncPersistWaitGroup.Go(func() error {
			defer func() { <-ctx.persistSem }()
			if err := s.insertManyWithTimeout(ctx.ctx, elog, docs); err != nil {
				if nerr := common.NonCtxCanceledErr(err); nerr != nil {
					stats.Record(ctx.ctx, metrics.ExtractError.M(1))
					elog.Errorf("insert extracted documents from tipsets: %s", err)
					return nerr
				}
			}

			return nil
		})
	} else {
		if err := s.insertManyWithTimeout(ctx.ctx, elog, docs); err != nil {
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

			res, err := s.extractJobWithTimeout(innerCtx, s.opts.Extract.StateJobTimeout, s.opts.Extract.JobRetry+1, ctx.L,
				fmt.Sprintf("regular state #%d", hi), func(jobCtx context.Context) (*extract.Res, error) {
					jctx := *ctx
					jctx.C = jobCtx
					jres := extract.NewRes(8, 0)
					if err := east.ExtractRegular(&jctx, jres, head); err != nil {
						return nil, err
					}
					return jres, nil
				})
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
		select {
		case pctx.persistSem <- struct{}{}:
		case <-originCtx.Done():
			return originCtx.Err()
		}

		pctx.asyncPersistWaitGroup.Go(func() error {
			defer func() { <-pctx.persistSem }()
			if err := s.insertMany(originCtx, ctx.L, docs); err != nil {
				stats.Record(originCtx, metrics.ExtractError.M(1))
				return fmt.Errorf("insert extracted documents from regular states: %w", err)
			}
			return nil
		})
	} else {
		if err := s.insertManyWithTimeout(originCtx, ctx.L, docs); err != nil {
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
	var dryEventsBlockstore blockstore.Blockstore
	if s.fullNode != nil {
		fullAPI := s.fullNode.GetAppropriateAPI()
		dryEventsBlockstore = blockstore.NewAPIBlockstore(&rawChainIO{api: fullAPI})
	}
	ectx, err := extract.NewCtx(ctx, s.dal, dlog, aset, latestDealID, dryOptions, s.fullNode, dryEventsBlockstore)
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
