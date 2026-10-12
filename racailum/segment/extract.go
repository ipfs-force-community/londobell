package segment

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opencensus.io/stats"

	"github.com/ipfs-force-community/londobell/metrics"

	"go.opencensus.io/trace"

	"github.com/filecoin-project/go-state-types/abi"
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
	ctx          context.Context
	actorSet     *actor.Set
	log          *zap.SugaredLogger
	asyncPersist asyncPersistTracker
	latestDealID int64
	persistSem   chan struct{}
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

// partPersistFailure 是「一次异步落库作业失败/超时」的 **part 归属**：哪个 part 的落库没成。
// 区间就是该 part 的 [FromEpoch,ToEpoch]，供上层把它转成阻断面（BlockedByPersistFailure）。
type partPersistFailure struct {
	FromEpoch abi.ChainEpoch
	ToEpoch   abi.ChainEpoch
	Err       error
}

// asyncPersistJob 是异步落库作业在 tracker 里的登记项：区间（part 归属）+ 完成状态与结果。
type asyncPersistJob struct {
	from     abi.ChainEpoch
	to       abi.ChainEpoch
	finished bool
	err      error
}

// asyncPersistTracker 追踪一个批次的异步落库作业，并保留每个作业的 part 归属。
//
// 它取代裸的 multierror.Group，理由（docs/cn/batch_failure_semantics.md §8.2 注入口 ②）：
// 裸 Group 只给出一个**合并**错误、无法归因到哪个 part，于是「落库超时」不能成为阻断面，
// 只能整批作废——正是主网 2026-10-12 事故（`error occurs in async persist ... context
// deadline exceeded`）放大成整批重跑的根因。带 part 归属后，失败 part 之前的连续前缀仍可
// 安全发布，失败处成为阻断面、下一轮重做。
//
// 并发口径：Go() 只在 part 循环（单 goroutine、顺序）里被调用，wait() 在循环结束之后才
// 调用，二者不重叠（multierror.Group 的 Wait 与 Go 不能重叠）；每个 job 的完成状态由 mu
// 保护，超时归因时读到的是前后一致的快照。
type asyncPersistTracker struct {
	mu    sync.Mutex
	group multierror.Group
	jobs  []*asyncPersistJob
}

// Go 登记并启动一次异步落库作业；[from,to] 是该作业的 part 归属，失败/超时时据此归因。
func (t *asyncPersistTracker) Go(from, to abi.ChainEpoch, fn func() error) {
	job := &asyncPersistJob{from: from, to: to}

	t.mu.Lock()
	t.jobs = append(t.jobs, job)
	t.mu.Unlock()

	t.group.Go(func() error {
		err := fn()
		t.mu.Lock()
		job.finished = true
		job.err = err
		t.mu.Unlock()
		return err
	})
}

// wait 等待所有作业结束（或超过 timeout），返回**按 part 归属**的失败清单。
//
// 语义相对旧 waitAsyncPersist（返回一个合并 error）的变化，正是本改动的主旨：
//   - 每个元素带自己的 part 区间，调用方据此把它当阻断面，而不是把整批作废；
//   - timeout <= 0 ⇒ 不设等待上限（等价旧行为：一直等 group 结束）；
//   - timeout > 0 且超时 ⇒ **仍未完成**的作业全部按「等待超时」归因（它们的 part 即阻断面）：
//     这些 part 的数据尚未落库成功，绝不能被前缀提交越过。
//
// Mongo 写入挂住时不能永久等待（否则整批静默停摆）；超时后残留的 goroutine 有上限
// （作业数），与旧实现同口径。
func (t *asyncPersistTracker) wait(timeout time.Duration) []partPersistFailure {
	timedOut := false
	if timeout <= 0 {
		t.group.Wait()
	} else {
		done := make(chan struct{})
		go func() {
			t.group.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(timeout):
			timedOut = true
		}
	}

	timeoutErr := fmt.Errorf("async persist wait timeout after %s", timeout)

	t.mu.Lock()
	defer t.mu.Unlock()

	var out []partPersistFailure
	for _, job := range t.jobs {
		switch {
		case job.err != nil:
			out = append(out, partPersistFailure{FromEpoch: job.from, ToEpoch: job.to, Err: job.err})
		case timedOut && !job.finished:
			out = append(out, partPersistFailure{FromEpoch: job.from, ToEpoch: job.to, Err: timeoutErr})
		}
	}
	return out
}

// partPersistFailuresToBlocking 把「按 part 归属的异步落库失败/超时」转成阻断面。
//
// 每个失败 part 转成一个 PartFailure（Kind=BlockedByPersistFailure）：**不写台账**——它是
// 「下一轮要重做」而非永久的洞。类别用 classifyErr 复用与「part 抽取失败」同一份判据
// （异步落库失败几乎必然是 transport 类 ⇒ 不可登记，故不会写成洞）。
func partPersistFailuresToBlocking(fails []partPersistFailure) []PartFailure {
	if len(fails) == 0 {
		return nil
	}

	out := make([]PartFailure, 0, len(fails))
	for _, pf := range fails {
		class, _ := classifyErr(pf.Err)
		out = append(out, PartFailure{
			FromEpoch: pf.FromEpoch,
			ToEpoch:   pf.ToEpoch,
			Class:     class,
			ErrDigest: truncateErrDigest(pf.Err.Error()),
			Kind:      BlockedByPersistFailure,
		})
	}
	return out
}

// asyncPersistOutcome 决定异步落库收尾的处置（纯函数，便于表驱动单测）：
//
//   - 无失败 ⇒ (nil, nil)：与旧实现「此时返回 nil error」一致，行为不变；
//   - 主路径（tmp=false）⇒ 转为阻断面 (blocking, nil)：失败 part 之前的连续前缀仍可发布，
//     失败处成为阻断面、下一轮重做——这是对主网 2026-10-12 事故的真修复；
//   - tmp 路径（tmp=true）⇒ (nil, fatalError)：**有意不改**（§8.11）。tmp 路径不走
//     commitPrefix，且 ExtractToTemporaryDB 会把段上界推到整批末，阻断面在那里无人消费、
//     反而会越过失败 part 推进边界 ⇒ 恢复旧行为：异步落库失败仍整批弃（fail-closed）。
func asyncPersistOutcome(tmp bool, fails []partPersistFailure) (blocking []PartFailure, fatal error) {
	if len(fails) == 0 {
		return nil, nil
	}
	if tmp {
		return nil, fmt.Errorf("error occurs in async persist: %s", summarizePartPersistFailures(fails))
	}
	return partPersistFailuresToBlocking(fails), nil
}

// summarizePartPersistFailures 把异步落库失败压成一行摘要（供 tmp 路径的错误正文）。
func summarizePartPersistFailures(fails []partPersistFailure) string {
	parts := make([]string, 0, len(fails))
	for _, f := range fails {
		parts = append(parts, fmt.Sprintf("[%d-%d:%v]", f.FromEpoch, f.ToEpoch, f.Err))
	}
	return strings.Join(parts, ",")
}

// partRange 返回 part 的 [起, 止] 高度；空 part 返回 (0,0)（调用方保证非空）。
func partRange(part []*common.LinkedTipSet) (abi.ChainEpoch, abi.ChainEpoch) {
	if len(part) == 0 {
		return 0, 0
	}
	return part[0].Height(), part[len(part)-1].Height()
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

// ExtractTipSets 抽取给定 tipset 区间的数据，返回**阻断清单**与 error。
//
// 阻断清单 = 已登记的洞 ∪ 失败 part（任何类别，含传输类 / 超时）。每个元素用 Kind 标注来源
// （BlockedByHole / BlockedByPersistFailure）；失败 part **不写台账**，它只是「下一轮要重做」。
//
// ★ 错误契约（阻断面语义）：
//   - 真正的致命错误（actor set / dealID 加载失败、台账写失败）⇒ 返回 (failures, err)，
//     err 非 nil。上层 `Segment.Extract` 照旧整批弃、不动水位。
//   - 某个 part 抽取失败（任何类别）⇒ **不再致命**：它的起始高度作为一条阻断面记入 failures，
//     返回 (failures, nil)，由上层只发布「第一个阻断面之前的连续前缀」。
//   - **异步落库失败/超时**（`Persist.Async=true`，含主网 2026-10-12 的
//     `error occurs in async persist ... context deadline exceeded`）⇒ 同样**不再致命**：
//     按 part 归属转成阻断面记入 failures（见 asyncPersistOutcome / partPersistFailuresToBlocking）。
//     **例外**：tmp 路径（tmp=true，ExtractToTemporaryDB）有意不改，仍整批弃（fail-closed）。
//
// **返回 nil error 不代表无阻断**——调用方必须检查 failures 是否为空，并据此计算「连续前缀」
// 后再决定发布到哪里（见 prefix_commit.go 的 commitPrefix / firstBlockingBoundary /
// prefixLenBeforeFirstHole）。这正是相对旧契约（有洞即报错、整批弃）的区别。
func (s *Segment) ExtractTipSets(ctx context.Context, tss []*common.LinkedTipSet, tmp bool) ([]PartFailure, error) {
	ctx, span := trace.StartSpan(ctx, "segment.ExtractTipSets")
	defer span.End()

	if len(tss) == 0 {
		return nil, nil
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
		// 非 tmp 路径复用与 tmp 同一套缓存（segment.Actorset + actorsetMu + storeActorSet），
		// 而不是每轮从最新 tipset 的状态根从头重建 actor set（实测单次重建 3分13秒~8分00秒，
		// 占一轮的 1/3~1/2）。缓存 key = 网络版本，另有有界兜底失效（见 acquireActorSet）。
		ts := tss[size-1]
		version := s.dal.GetNetworkVersion(ctx, ts.Height())

		var why actorSetInvalidation
		aset, why, err = acquireActorSet(version, ts.Height(), func() (*actor.Set, error) {
			return actor.NewSet(ctx, s.dal.StateManager, ts, tmp)
		})
		if err != nil {
			return nil, err
		}
		logActorSetOutcome(elog, why, version, ts.Height())

		latestDealID, err = s.GetLatestDealID(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		ts := tss[0]
		version := s.dal.GetNetworkVersion(ctx, ts.Height())

		// 「版本变化」与「有界兜底失效」都要重建，但只有前者才套昂贵 epoch 守卫（保留原行为）：
		// 升级边界恰落在昂贵 epoch 时，宁可沿用旧集合（同版本内本来也是复用的）也不做那次重载。
		// 有界兜底是例行维护性重建，不受该守卫影响。
		if cacheWhy := actorSetInvalidationOf(loadActorSet(), version, ts.Height()); cacheWhy == actorSetVersionChanged &&
			s.opts.Extract.ExtractOptions.SkipExpensiveEpoch && tipset.IsExpensive(ctx, s.dal.StateManager, ts) {
			// TODO: extract simple invoc results here
			elog.Warn("ignore expensive epoch actor load")
			// 保持原行为：不重建，aset 仍为 nil。
		} else {
			var why actorSetInvalidation
			aset, why, err = acquireActorSet(version, ts.Height(), func() (*actor.Set, error) {
				return actor.NewSet(ctx, s.dal.StateManager, ts, tmp)
			})
			if err != nil {
				return nil, err
			}
			if why == actorSetVersionChanged {
				elog.Infof("reload new version: %s actor", version)
			}
			logActorSetOutcome(elog, why, version, ts.Height())
		}
	}

	pctx := &persistCtx{
		ctx:          ctx,
		actorSet:     aset,
		log:          elog,
		latestDealID: latestDealID,
		persistSem:   make(chan struct{}, 2),
	}

	// 把 part 循环抽到 partRunner：依赖全部经字段注入，生产路径注入真实的
	// extractPart / 台账写入 / 护栏取值；单测注入假实现，从而在不连链、不连 mongo 的
	// 前提下验证「分类 → 登记或标记阻断面 → 继续（只抽取不落库）」这条契约：
	//   - 可登记类：写台账成为「洞」；
	//   - 不可登记类（传输/超时/未知）与未开启类：不写台账，作为「失败 part 阻断面」；
	// 二者都进阻断清单返回，发布决策（只推第一个阻断面之前的前缀）交给上层。
	runner := &partRunner{
		partSizeLimit:  s.opts.Extract.TipSetPartSizeLimit,
		skippable:      s.opts.Extract.SkippableErrorClasses,
		maxUnpublished: s.opts.Extract.MaxUnpublishedRange,
		log:            elog,
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			return s.extractPart(pctx, part, tmp, persist)
		},
		record: func(pf PartFailure) error {
			return s.recordSkippedPart(ctx, elog, pf)
		},
		unpublishedRange: func() (abi.ChainEpoch, bool) {
			return s.unpublishedRange(ctx, tss[size-1])
		},
	}

	failures, rerr := runner.run(tss)

	// 致命错误（不可登记的 part 失败 / 台账写失败）：保持改动前的现状——立即整批作废、
	// 不动水位（上层据此整批弃）。
	if rerr != nil {
		return failures, rerr
	}

	// 异步落库收尾：失败/超时按 part 归属（§8.2 注入口 ②）。
	// 主路径转为阻断面（失败 part 之前的连续前缀照常发布）；tmp 路径保持旧行为（整批弃）——
	// 判定与理由见 asyncPersistOutcome。
	blocking, aerr := asyncPersistOutcome(tmp, pctx.asyncPersist.wait(s.opts.Persist.WaitTimeout))
	if aerr != nil {
		return failures, aerr
	}
	failures = append(failures, blocking...)

	if len(failures) == 0 {
		elog.Info("all parts done")
	} else {
		elog.Warnw("all parts done with blocking boundaries (publish decision deferred to Segment.Extract)",
			"blockedBy", summarizeFailures(failures))
	}

	// ★ 契约：返回的 failures = 阻断清单 = 已登记的洞 ∪ 失败 part ∪ 异步落库失败 part。
	// 返回 nil error 只表示「循环本身没有致命失败」，不代表无阻断——由上层 Segment.Extract
	// 按「连续前缀」决定发布到哪里（见 prefix_commit.go 的 commitPrefix）。唯一仍会在此
	// return error 的是「台账写失败」那条铁律（不可登记的 part 失败不再致命）。
	return failures, nil
}

// partRunner 承载 ExtractTipSets 的 part 循环。依赖注入式的字段让它可被单测覆盖，
// 而不必构造真实的 Segment / 链 / mongo。之所以单独成类型而不用 Segment 上的方法：
// 这样循环的控制流（分类、登记、继续、护栏、返回登记清单）能被独立、确定性地测到。
type partRunner struct {
	partSizeLimit  int
	skippable      []string
	maxUnpublished abi.ChainEpoch
	log            *zap.SugaredLogger

	// extractPart 抽一个 part；persist=false 表示只抽取、不落库。
	extractPart func(part []*common.LinkedTipSet, persist bool) error
	// record 把失败登记进台账；返回 error 时该 part 按「不可登记」处理（铁律）。
	record func(pf PartFailure) error
	// unpublishedRange 取「链头 − 已发布水位」；ok=false 表示取不到（护栏不生效）。
	unpublishedRange func() (abi.ChainEpoch, bool)
}

// classEnabled 判断某个错误类别是否被配置为「可登记并继续」。
// 空配置 = 完全维持现状（不登记、不跳过）。
func classEnabled(class string, skippable []string) bool {
	for _, c := range skippable {
		if c == class {
			return true
		}
	}
	return false
}

// guardExceeded 判定护栏是否已触发。护栏只在「已经进入只抽取不落库模式」后才被询问；
// 且取不到水位、或阈值非正时都不拦（宁可停在这里，也不拿不确定的值提前停手）。
func (r *partRunner) guardExceeded() bool {
	if r.maxUnpublished <= 0 || r.unpublishedRange == nil {
		return false
	}
	v, ok := r.unpublishedRange()
	if !ok {
		return false
	}
	return v > r.maxUnpublished
}

func (r *partRunner) run(tss []*common.LinkedTipSet) ([]PartFailure, error) {
	size := len(tss)
	if size == 0 {
		return nil, nil
	}

	limit := r.partSizeLimit
	if limit <= 0 {
		limit = size
	}
	partTotal := (size + limit - 1) / limit

	var failures []PartFailure
	tsDone := 0
	partDone := 0
	for tsDone < size {
		start := tsDone
		end := start + limit
		if end > size {
			end = size
		}
		part := tss[start:end]

		// 一旦有 part 失败被登记，后续 part 只抽取、不落库：坏 part 之后的数据一律
		// 不得写进 mongo，否则客户端用显式区间就能读到「跨洞的两侧数据」（聚合器不按
		// 水位拦截区间查询，见 docs/cn/batch_failure_semantics.md §8.3）。
		persist := len(failures) == 0

		// 护栏（§8.5）：已在「只抽取不落库」模式下且未发布区间超阈值 ⇒ 停止继续往后抽，
		// 否则洞不愈合时 bell 会一路抽到链头、mongo 里堆积不可见文档。
		if !persist && r.guardExceeded() {
			r.log.Warnw("stop extracting further parts: unpublished range would exceed MaxUnpublishedRange",
				"max-unpublished", int64(r.maxUnpublished), "done-parts", partDone, "part-total", partTotal)
			break
		}

		err := r.extractPart(part, persist)
		if err != nil {
			pf, recordable := classifyPartFailure(part, err)
			if recordable && classEnabled(pf.Class, r.skippable) {
				// 可登记类别且已开启：写台账，成为一个「洞」（既有语义不变）。
				// 铁律：台账写不进去就不允许跳过。写失败 ⇒ 整批作废（真正的致命错误，保持现状）。
				if rerr := r.record(pf); rerr != nil {
					return failures, fmt.Errorf("extract part %d/%d failed and ledger write failed (range %s, completed %d/%d): %v; original error: %w",
						partDone+1, partTotal, common.FormatTipSetEpochRange(part), partDone, partTotal, rerr, err)
				}
				pf.Kind = BlockedByHole
				failures = append(failures, pf)
				r.log.Warnw("part failure recorded as hole, continuing with persist disabled",
					"class", pf.Class, "from", int64(pf.FromEpoch), "to", int64(pf.ToEpoch), "cid", pf.CID)
			} else {
				// 不可登记（传输类 / 超时 / 未知）或该类未开启：**不写台账**——它是「下一轮要重做」，
				// 不是永久的洞；但它的起始高度就是一个阻断面：本轮从它之前截断发布、水位停在前缀末，
				// 之后即使还有已成功的 part 也绝不跨越它发布（见 commitPrefix / firstBlockingBoundary）。
				// 不再整批作废——这正是本段要消灭「一个坏 part 拖垮整批」的落点。
				pf.Kind = BlockedByPersistFailure
				failures = append(failures, pf)
				r.log.Warnw("part failure is a blocking boundary (not recorded), continuing with persist disabled",
					"class", pf.Class, "from", int64(pf.FromEpoch), "to", int64(pf.ToEpoch), "err", pf.ErrDigest)
			}
		} else {
			r.log.Infow("part done", "done-parts", partDone+1, "done-tss", tsDone+len(part), "persisted", persist)
		}

		tsDone += len(part)
		partDone++
	}

	// ★ 契约：阻断清单（已登记的洞 ∪ 失败 part）**不再**当 error 返回。返回 nil error 只表示
	// 「循环本身没有致命失败」，不代表无阻断——由上层 Segment.Extract 按「连续前缀」决定发布到
	// 哪里。唯一仍会在循环里 return error 的是「台账写失败」那条铁律。
	return failures, nil
}

// unpublishedRange 计算「链头 − 已发布水位」。
//
// head 取当前批的最高 tipset：批区间是 (final_height, 链头]，所以它是本轮能看到的最高高度。
// 水位从 FinalHeight 读（读库 s.rdb）。任何一步取不到（读库未配置、查询失败）都返回
// ok=false——护栏宁可不生效，也不拿不确定的值去拦（详见提交说明里的不确定项）。
func (s *Segment) unpublishedRange(ctx context.Context, head *common.LinkedTipSet) (abi.ChainEpoch, bool) {
	if head == nil || s.rdb == nil {
		return 0, false
	}
	fh, err := s.GetFinalHeight(ctx)
	if err != nil {
		return 0, false
	}
	if head.Height() <= fh {
		return 0, true
	}
	return head.Height() - fh, true
}

func (s *Segment) extractPart(ctx *persistCtx, part []*common.LinkedTipSet, tmp bool, persist bool) error {
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

	if err := s.persistPartDocs(ctx, elog, part, docs, persist); err != nil {
		return err
	}

	// 只抽取不落库时，regular states 同样不抽取、不落库：它的产物也是要写 mongo 的数据，
	// 一并丢弃才能保证「坏 part 之后的可见区间为空」。
	if !persist {
		return nil
	}

	// temporary db don't need to store state datas
	if tmp {
		return nil
	}

	// reset context
	ectx.C = ctx.ctx
	for rhi := range regulars {
		if rheads := regulars[rhi]; len(rheads) > 0 {
			if err := s.extractRegularStates(ectx, ctx, part, rheads); err != nil {
				return fmt.Errorf("#%d regular heads: %w", rhi, err)
			}
		}
	}

	return nil
}

// persistPartDocs 把 part 的抽取结果落库。
//
// persist=false ⇒ 只记一条日志、直接丢弃（既不调 insertManyWithTimeout、也不进
// asyncPersistTracker）——这是「坏 part 之后只抽取、不落库」的落点，目的是让写进
// mongo 的可见区间始终连续（docs/cn/batch_failure_semantics.md §8.3）。
//
// 异步路径把 part 的 [起,止] 一并登记进 tracker：落库失败/超时时，交回上层的失败清单带
// part 归属，从而能成为阻断面而不是整批作废（§8.2 注入口 ②）。
func (s *Segment) persistPartDocs(ctx *persistCtx, elog *zap.SugaredLogger, part []*common.LinkedTipSet, docs [][]common.Document, persist bool) error {
	if !persist {
		elog.Infow("part extracted but not persisted (persist=false: skipped after a recorded failure)")
		return nil
	}

	if s.opts.Persist.Async {
		select {
		case ctx.persistSem <- struct{}{}:
		case <-ctx.ctx.Done():
			return ctx.ctx.Err()
		}

		from, to := partRange(part)
		ctx.asyncPersist.Go(from, to, func() error {
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
		return nil
	}

	if err := s.insertManyWithTimeout(ctx.ctx, elog, docs); err != nil {
		return fmt.Errorf("insert extracted documents from tipsets: %w", err)
	}

	return nil
}

func (s *Segment) extractRegularStates(ctx *extract.Ctx, pctx *persistCtx, part []*common.LinkedTipSet, heads []*common.ActorHead) error {
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

		from, to := partRange(part)
		pctx.asyncPersist.Go(from, to, func() error {
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
