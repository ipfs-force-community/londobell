package segment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opencensus.io/stats"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mdict"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

var log = logging.Logger("segment")

type ActorSet struct {
	Version network.Version
	Set     *actor.Set

	// 有界兜底失效的账面（见 actorSetReuseBudget / actorSetEpochWindow）：
	Reuses  int            // 自构建以来被复用的轮数；每复用一次 +1，重建时归零
	BuiltAt abi.ChainEpoch // 构建该集合时所用的 tipset 高度
}

var (
	Actorset   *ActorSet
	actorsetMu sync.RWMutex
)

// 有界兜底失效参数（两个条件取先到者）：
//
//	actorSetReuseBudget —— 同一网络版本内，缓存最多被复用多少轮；
//	actorSetEpochWindow —— 距构建的链上高度推进上限。
//
// 为什么需要：缓存 key 取网络版本（见 acquireActorSet），它覆盖了 actor code 变更的真实来源
// ——网络升级迁移（实测不同版本会改 code、同版本内几乎不变）。但「actor 仍在集合里、code 变了」
// 不会产生 MISS：actor.Set.LookupMethodInfo 只在集合里查不到该地址时才回落 stm.LoadActor 并
// 把 code 回写（actor/set.go:169-186），集合里查得到就直接用旧 code ⇒ 静默出错的窗口。
// 下面两个窗口把「沿用旧 code 的最长时间」钉住，是一条**保险**，正常情况下不触发。
//
// 取值依据（实测：单次重建 3分13秒~8分00秒；一轮抽取 2分29秒~15分55秒、21~25 tipset/轮）：
//   - 兜底窗口越大，重建越少、分摊开销越低，但沿用旧 code 的最长时间越长。
//   - 升级间隔以周计 ⇒ 兜底窗口只要远小于升级间隔即安全：actorSetEpochWindow=2000 epoch
//     （≈16.7h 链时）远小于任何升级间隔。
//   - actorSetReuseBudget=30 轮（按实测 ~22 tipset/轮 ≈ 660 epoch ≈ 5.5h 链时）在正常节奏下
//     先生效 ⇒ 约每 5~6 小时重建一次，分摊到每轮 ≈ 6~16 秒（一轮的 2~8%），远优于「每轮重建」。
//   - 轮次预算同时兜住「轮很小、很频繁」这种 epoch 窗口迟迟不生效的形态（此时按轮数兜底）。
const (
	actorSetReuseBudget = 30
	actorSetEpochWindow = 2000
)

// actorSetInvalidation 说明当前缓存相对期望版本为何不可复用（决定是否重建）。
type actorSetInvalidation int

const (
	actorSetValid           actorSetInvalidation = iota // 命中缓存，可复用
	actorSetMissing                                     // 无缓存
	actorSetVersionChanged                              // 网络版本变化（升级迁移）
	actorSetBudgetExhausted                             // 复用轮数达上界（有界兜底）
	actorSetWindowExceeded                              // 距构建已超 epoch 窗口（有界兜底）
)

func (w actorSetInvalidation) String() string {
	switch w {
	case actorSetMissing:
		return "missing"
	case actorSetVersionChanged:
		return "version-changed"
	case actorSetBudgetExhausted:
		return "budget-exhausted"
	case actorSetWindowExceeded:
		return "epoch-window-exceeded"
	default:
		return "valid"
	}
}

// actorSetSnapshot 是缓存条目的只读快照。Reuses/BuiltAt 会在锁内被改动，故禁止在锁外直接读
// Actorset 的字段——一律经本快照，避免 data race。
type actorSetSnapshot struct {
	exists  bool
	version network.Version
	reuses  int
	builtAt abi.ChainEpoch
}

// actorSetSnapshotLocked 取缓存快照；调用方必须已持有 actorsetMu（读或写锁均可）。
func actorSetSnapshotLocked() actorSetSnapshot {
	if Actorset == nil {
		return actorSetSnapshot{}
	}
	return actorSetSnapshot{
		exists:  true,
		version: Actorset.Version,
		reuses:  Actorset.Reuses,
		builtAt: Actorset.BuiltAt,
	}
}

// loadActorSet 在锁内取缓存快照（无缓存时 exists=false）。注意：它返回的是**快照**而非原指针
// ——原实现返回原指针，会让调用方在锁外读可变字段（Reuses），是与并发写并发的 data race。
func loadActorSet() actorSetSnapshot {
	actorsetMu.RLock()
	defer actorsetMu.RUnlock()
	return actorSetSnapshotLocked()
}

// actorSetInvalidationOf 是复用判定的纯函数（无全局状态、无锁），便于表驱动测试各失效边界。
// 只有「缓存存在 && 版本一致 && 复用轮数未达预算 && 距构建未超 epoch 窗口」同时成立才可复用。
func actorSetInvalidationOf(snap actorSetSnapshot, want network.Version, nowEpoch abi.ChainEpoch) actorSetInvalidation {
	switch {
	case !snap.exists:
		return actorSetMissing
	case snap.version != want:
		return actorSetVersionChanged
	case snap.reuses >= actorSetReuseBudget:
		return actorSetBudgetExhausted
	case nowEpoch-snap.builtAt >= actorSetEpochWindow:
		return actorSetWindowExceeded
	default:
		return actorSetValid
	}
}

// acquireActorSet 为本次抽取取一个 actor set（tmp 与非 tmp 路径共用这一套，不另起第二套缓存）：
//
//   - 命中缓存 ⇒ 复用并把复用计数 +1（省下 3~8 分钟的「状态树 384 万条 + init address_map 386 万条」
//     从头重建）；这正是本次改动的目的：非 tmp 路径不再每轮重建。
//   - 未命中 ⇒ 调 build 重建，把新集合写回缓存（计数归零、记录构建高度 builtAt）。
//
// 返回的 why 说明**本次为何走到重建**（actorSetValid 表示命中复用），供调用方打可区分的日志。
//
// 并发：判定（读版本 + 判预算/窗口）与「占用一次复用额度」同处一把 actorsetMu 写锁内，原子。
// 重建放在锁外——单次重建 3~8 分钟，占着全局锁会让其它抽取轮次整段阻塞；并发重建最多多做一次
// 无用功（后写者的集合同样正确），不会产出错误结果。
func acquireActorSet(want network.Version, builtAt abi.ChainEpoch, build func() (*actor.Set, error)) (*actor.Set, actorSetInvalidation, error) {
	actorsetMu.Lock()
	why := actorSetInvalidationOf(actorSetSnapshotLocked(), want, builtAt)
	if why == actorSetValid {
		Actorset.Reuses++ // 与上面的判定同处一把写锁内，读-改-写原子
		set := Actorset.Set
		actorsetMu.Unlock()
		return set, actorSetValid, nil
	}
	actorsetMu.Unlock()

	set, err := build()
	if err != nil {
		return nil, why, err
	}

	storeActorSet(&ActorSet{Version: want, Set: set, BuiltAt: builtAt})
	return set, why, nil
}

// logActorSetOutcome 打出 actor set 缓存的可区分日志：
//   - 命中复用 ⇒ "actor set reused"；
//   - 因有界兜底（预算 / epoch 窗口）重建 ⇒ "actor set cache invalidated, rebuilding"；
//   - 其余重建（无缓存 / 版本变化）不额外打日志——构建本身由 actor.NewSet 的
//     "actor set load" / "actor set loaded" 这对 INFO 覆盖（构建耗时靠它量）。
//
// 注意：这里不触碰 LookupMethodInfo 的 "code not found ... exists in init state" WARN 级别
// ——那是另一件事，本次不改。
func logActorSetOutcome(l *zap.SugaredLogger, why actorSetInvalidation, version network.Version, epoch abi.ChainEpoch) {
	switch why {
	case actorSetValid:
		l.Infow("actor set reused", "key", version, "epoch", epoch)
	case actorSetBudgetExhausted, actorSetWindowExceeded:
		l.Infow("actor set cache invalidated, rebuilding", "key", version, "epoch", epoch, "reason", why.String())
	}
}

func ClearActorSet() {
	actorsetMu.Lock()
	Actorset = nil
	actorsetMu.Unlock()
}

func storeActorSet(as *ActorSet) {
	actorsetMu.Lock()
	Actorset = as
	actorsetMu.Unlock()
}

func NewActorSet(ctx context.Context, stm common.StateManager, ts *common.LinkedTipSet, allowNilChild bool) (*ActorSet, error) {
	set, err := actor.NewSet(ctx, stm, ts, allowNilChild)
	if err != nil {
		return nil, fmt.Errorf("new actor set failed: %w", err)
	}

	version := stm.GetNetworkVersion(ctx, ts.Height())

	return &ActorSet{
		Version: version,
		Set:     set,
		BuiltAt: ts.Height(),
	}, nil
}

// Anchor contains most significant info about a tipset
type Anchor struct {
	Epoch abi.ChainEpoch
	TSK   types.TipSetKey
	State cid.Cid
}

// Is checks if the anchor is the given tipset
func (a *Anchor) Is(ts *common.LinkedTipSet) bool {
	return a.TSK == ts.Key() && a.State == ts.State()
}

// New attempts to construct a *Segment
func New(ctx context.Context, name string, opts Options, aggopt aggregate.Options, mgr *Manager, cs common.ChainStore, stm common.StateManager, fullNode common.FullNodeApiGetter) (*Segment, error) {
	bound, bhas, err := mgr.LoadBoundary(name)
	if err != nil {
		return nil, fmt.Errorf("load boundary: %w", err)
	}

	if !bhas {
		return nil, fmt.Errorf("boundary not found")
	}

	info, ihas, err := mgr.LoadInfo(name)
	if err != nil {
		return nil, fmt.Errorf("load info: %w", err)
	}

	if !ihas {
		return nil, fmt.Errorf("info not found")
	}

	multiWdocs := &mgoutil.MultiDB{}
	dicts := &mdict.Dicts{}

	for _, write := range info.DSN.NewWrite {
		wcli, err := mgoutil.Connect(ctx, write)
		if err != nil {
			return nil, fmt.Errorf("connect to write db: %w", err)
		}

		wdb := wcli.Database(name)

		wdoc, err := mgoutil.NewMgoDocDB(ctx, wcli, wdb)
		if err != nil {
			return nil, fmt.Errorf("construct write doc db: %w", err)
		}

		dict, err := mdict.NewDict(wdb)
		if err != nil {
			return nil, fmt.Errorf("construct Dict: %w", err)
		}

		err = multiWdocs.SetDbs(wdoc)
		if err != nil {
			return nil, fmt.Errorf("multiwdocs setdbs: %w", err)
		}

		err = dicts.SetDicts(dict)
		if err != nil {
			return nil, fmt.Errorf("dicts setdicts: %w", err)
		}
	}

	//rdoc := wdoc
	var rdoc common.DocumentDB
	if info.DSN.Read != "" {
		rcli, err := mgoutil.Connect(ctx, info.DSN.Read)
		if err != nil {
			return nil, fmt.Errorf("connect to read db: %w", err)
		}

		rdoc, err = mgoutil.NewMgoDocDB(ctx, rcli, rcli.Database(name))
		if err != nil {
			return nil, fmt.Errorf("construct read doc db: %w", err)
		}
	}

	agg, err := aggregate.New(aggopt, multiWdocs)
	if err != nil {
		return nil, err
	}

	seg := &Segment{
		name: name,
		opts: opts,
		db:   multiWdocs,
		rdb:  rdoc,
		agg:  agg,
		mgr:  mgr,
	}

	seg.bound.Boundary = bound

	seg.dal.ChainStore = cs
	seg.dal.ChainDict = dicts
	seg.dal.StateManager = stm
	seg.fullNode = fullNode

	return seg, nil
}

// Segment is one partition of the structrued data extracted from the chain
type Segment struct {
	name string

	opts Options
	db   common.DocumentDB
	rdb  common.DocumentDB
	agg  *aggregate.Aggregator
	mgr  *Manager

	headNotify chan *types.TipSet
	fullNode   common.FullNodeApiGetter
	bound      struct {
		sync.RWMutex
		Boundary
	}

	dal struct {
		common.ChainStore
		common.ChainDict
		common.StateManager
	}
}

// Incoming responds to a new heaviest tipset
func (s *Segment) Incoming(ctx context.Context, ts *types.TipSet) {
	select {
	case s.headNotify <- ts:

	default:

	}

	return

}

// Run starts a watching loop for incoming tipsets
func (s *Segment) Run(ctx context.Context) {
	log.Info("start head watching loop start")
	defer log.Info("stop head watching loop")

	for {
		select {
		case <-ctx.Done():
			return

		case ts := <-s.headNotify:
			if err := s.Extract(ctx, ts); err != nil {
				log.Errorw("failed to handle inocoming tipset", "tsk", ts.Key(), "tsh", ts.Height(), "err", err.Error())
			}
		}
	}
}

// RoundOutcome 描述一次抽取轮次（Segment.ExtractRound）的结果，供上层（RaCailum.Run 循环）
// 决定是否轮间退避。
type RoundOutcome struct {
	// Truncated 为 true 表示本轮被阻断面（已登记的洞 ∪ 失败 part ∪ 异步落库失败 part）截断：
	// 水位只前移到阻断面之前的连续前缀，整批并未抽完。**这不是错误**（数据是对的、算部分成功），
	// 但下一轮必然在同一阻断面重做整段，故 Run 循环应据此退避——否则每 MinPeriod（约 20s）
	// 白抽一遍整段（实测 cali 每轮白跑 826 个高度）。
	Truncated bool
	// BlockedAt 是本轮最早阻断面的起始高度（Truncated 时才有效）；仅供日志观测。
	BlockedAt abi.ChainEpoch
}

// roundOutcomeFromFailures 由阻断清单算本轮结果（纯函数，便于单测）：
// 有阻断 ⇒ Truncated=true 并带上最早阻断面的起点；无阻断 ⇒ Truncated=false。
//
// 为什么「有阻断」就等于「被截断」：failures 的元素都来自批内的 part，其 FromEpoch 必 ≤ 批末，
// 故 prefixLenBeforeFirstHole 必 < len(tss)——本轮绝不可能把整批发布完（见 prefix_commit.go）。
func roundOutcomeFromFailures(fails []PartFailure) RoundOutcome {
	b, ok := firstBlockingBoundary(fails)
	if !ok {
		return RoundOutcome{BlockedAt: -1}
	}
	return RoundOutcome{Truncated: true, BlockedAt: b.FromEpoch}
}

// Extract attempts to extract data between given tipset and the hi-bound of the segment.
//
// 只返回 error（丢弃 RoundOutcome）——供不在乎「本轮是否被阻断面截断」的调用方使用
// （对外的手动抽取入口等）。需要据截断决定轮间退避的调用方（RaCailum.Run）请用 ExtractRound。
func (s *Segment) Extract(ctx context.Context, rawts *types.TipSet) error {
	_, err := s.ExtractRound(ctx, rawts)
	return err
}

// ExtractRound 是抽取的实现体：除 error 外还返回本轮的 RoundOutcome（是否被阻断面截断）。
//
// 阻断轮（Truncated）返回 nil error：它是部分成功、数据正确，只是水位停在阻断面之前；
// 上层 Run 循环据此退避（避免每 MinPeriod 白抽一遍），**不要**把它当持久化失败记日志
// ——运维告警是按 "failed to persist tipset" 这行做的（见 ra.go）。
func (s *Segment) ExtractRound(ctx context.Context, rawts *types.TipSet) (RoundOutcome, error) {
	ctx, span := trace.StartSpan(ctx, "segment.Extract")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(rawts.Height())))
	defer span.End()

	tsk := rawts.Key()
	tsh := rawts.Height()

	start := time.Now()
	defer func() {
		log.Infow("extract done", "raw-tsk", tsk, "raw-tsh", tsh, "elapsed", time.Now().Sub(start).String())
	}()

	s.bound.Lock()
	defer s.bound.Unlock()

	lo := s.bound.Lo.Epoch
	if tsh <= lo {
		return RoundOutcome{}, fmt.Errorf("%s is not belong to this segment", common.FormatTipSet(rawts))
	}

	hi := s.bound.Hi.Epoch
	if tsh > hi+s.opts.Extract.MaxBackward {
		return RoundOutcome{}, fmt.Errorf("%s is too far away from current upper bound @%d", common.FormatTipSet(rawts), hi)
	}

	if hi+s.opts.Extract.MinPeriod+s.opts.Extract.Confidence >= tsh {
		return RoundOutcome{}, nil
	}

	tipsets, err := ExtractLinkedTipSets(s.dal.ChainStore, rawts, &hi, false)
	if err != nil {
		return RoundOutcome{}, err
	}

	if !s.bound.Hi.Is(tipsets[0]) {
		return RoundOutcome{}, fmt.Errorf("current segment is not the ancestor (%s) of the incoming chain: %s", s.bound.Hi.TSK, tipsets[0])
	}

	for i := 1; i < len(tipsets); i++ {
		tipsets[i].Parent = tipsets[i-1].TipSet
	}

	tipsets = tipsets[1:]

	confidentEpoch := tsh - s.opts.Extract.Confidence
	tssize := len(tipsets)
	for ; tssize > 0; tssize-- {
		if tipsets[tssize-1].Height() <= confidentEpoch {
			break
		}
	}

	tipsets = tipsets[:tssize]
	if len(tipsets) == 0 {
		return RoundOutcome{}, nil
	}

	fails, ferr := s.ExtractTipSets(ctx, tipsets, false)
	if ferr != nil {
		return RoundOutcome{}, ferr // 真正的致命错误（actor set / dealID 加载、台账写失败等）：整批弃、不动水位
	}

	// 只把「第一个阻断面之前的连续前缀」交给聚合与水位推进（★ 红线，见 prefix_commit.go / §8.3）。
	// 无阻断时 prefix = 整批，行为与改动前一致；阻断面在批首（prefixLen==0）时不发布、不动水位。
	// 阻断面在批中 ⇒ 只推前缀末；两种情况**都不算致命失败**（部分成功 / 下一轮从阻断面续跑）。
	if perr := commitPrefix(ctx, tipsets, fails, prefixCommitDeps{
		aggregate:        func(c context.Context, p []*common.LinkedTipSet) error { return s.Aggregate(c, p) },
		saveFinalHeight:  func(c context.Context, h *common.LinkedTipSet) error { return s.SaveFinalHeight(c, h) },
		saveStateHeight:  func(c context.Context, h *common.LinkedTipSet) error { return s.SaveStateFinalHeight(c, h) },
		updateBoundary:   func(c context.Context, h *common.LinkedTipSet) error { return s.updateBoundary(c, h, nil) },
		onlyExtractState: s.opts.Extract.OnlyExtractState,
		saveStateEnabled: s.opts.Extract.ExtractOptions.EnabelExtract.EnableExtractState,
	}); perr != nil {
		// 发布本身落地失败（Aggregate / SaveFinalHeight / updateBoundary 报错）：真正的致命错误。
		return RoundOutcome{}, perr
	}

	// 本轮是否被阻断面截断（存在阻断 ⇒ 水位只推到阻断面之前）。这不是错误：数据对、算部分成功；
	// 但下一轮会在同一阻断面重做整段，上层 Run 循环据此退避（见 RoundOutcome）。
	outcome := roundOutcomeFromFailures(fails)

	// ★ 台账 heal 闭环（healed_at 的唯一写入路径；§8.4 / §8.9 验收第 4 条）：发布成功后
	// 对账一次，把「已被本实例已发布水位越过末端」的洞标记为已修复。
	//
	// 位置理由：只有在 commitPrefix 返回 nil 且本轮真的前移了水位时（prefixLen > 0），本轮的
	// 发布才真正落地（Aggregate + SaveFinalHeight/SaveStateFinalHeight + updateBoundary 全部成功），
	// 此刻「本实例已发布的最终水位」就是刚发布的前缀末高度。洞的愈合恰恰发生在**干净轮次**——本轮整批成功、
	// 前缀 = 整批，水位一举越过此前登记的洞末端——所以本对账必须**每轮**都尝试，而不是只在
	// 「本轮有洞」时尝试，否则洞永远等不到被销项。（前缀高度用与 commitPrefix 同一份纯函数
	// prefixLenBeforeFirstHole 复算取得，不改 prefix_commit.go 的签名。）
	//
	// 「每轮最多一次」：Extract 每次调用只会执行到这里一次，此处也只调一次
	// ReconcileHealedParts（内部是一条 updateMany），绝不在 partRunner 里按 part 调。
	//
	// 「失败不影响批次」：发布已完成、数据是对的，对账只是台账销项；失败只记一条日志，
	// 绝不返回 error（与「台账写不进去就不允许跳过」那条铁律不同——那针对的是危险的「跳过」
	// 动作，销项失败不危险，不该把已正确的一批判成失败、拖住水位）。
	if prefixLen := prefixLenBeforeFirstHole(tipsets, fails); prefixLen > 0 {
		// 只有「本轮真的前移了本实例负责的水位」才谈得上证明：statebell 在 EnableExtractState
		// 关闭时不会前移任何水位（见 commitPrefix），此时前缀末并非水位，不能拿它对账。
		if !s.opts.Extract.OnlyExtractState || s.opts.Extract.ExtractOptions.EnabelExtract.EnableExtractState {
			watermark := tipsets[prefixLen-1].Height()
			if herr := s.ReconcileHealedParts(ctx, watermark, healClassesFor(s.opts.Extract.OnlyExtractState)); herr != nil {
				log.Warnw("reconcile healed parts failed; batch unaffected",
					"watermark", int64(watermark), "err", herr.Error())
			}
		}
	}

	return outcome, nil
}

func (s *Segment) ExtractToTemporaryDB(ctx context.Context, rawts *types.TipSet) error {
	log.Infow("extract tipset to temporary db", "tipset.height", rawts.Height(), "tipset.cids", rawts.Key())
	ctx, span := trace.StartSpan(ctx, "segment.ExtractToTemporaryDB")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(rawts.Height())))
	defer span.End()

	tmp := true

	s.bound.Lock()
	defer s.bound.Unlock()

	tsh := rawts.Height()
	hi := s.bound.Hi.Epoch
	if hi >= tsh {
		return fmt.Errorf("hi %v of tmp exceeds rawts height %v", hi, tsh)
	}

	tipsets, err := ExtractLinkedTipSets(s.dal.ChainStore, rawts, &hi, tmp)
	if err != nil {
		return err
	}

	for i := 1; i < len(tipsets); i++ {
		tipsets[i].Parent = tipsets[i-1].TipSet
	}

	// extract (hi, rawts]
	tipsets = tipsets[1:]

	if _, err := s.ExtractTipSets(ctx, tipsets, tmp); err != nil {
		return err
	}
	if err := s.updateBoundary(ctx, tipsets[len(tipsets)-1], nil); err != nil {
		return err
	}

	return nil

}

func (s *Segment) GetTmpFinalTipSet(ctx context.Context) (*types.TipSet, error) {
	tmpFinalTipSet, err := s.dal.LoadTipSet(ctx, s.ReadBoundary().Hi.TSK)
	if err != nil {
		return nil, err
	}

	return tmpFinalTipSet, nil
}

func (s *Segment) GetTipSetByTSk(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	ts, err := s.dal.LoadTipSet(ctx, tsk)
	if err != nil {
		return nil, err
	}

	return ts, nil
}

// Aggregate tries to do aggregationg with given tipsets
func (s *Segment) Aggregate(ctx context.Context, tss []*common.LinkedTipSet) error {
	err := s.agg.Aggregate(ctx, tss)
	if err != nil {
		return err
	}

	return nil
}

func (s *Segment) updateBoundary(ctx context.Context, hi, lo *common.LinkedTipSet) error {
	_, span := trace.StartSpan(ctx, "segment.updateBoundary")
	defer span.End()

	prev := s.bound.Boundary

	if hi != nil {
		stats.Record(ctx, metrics.UpperBoundary.M(int64(hi.Height())))
		s.bound.SetHi(hi)
	}

	if lo != nil {
		stats.Record(ctx, metrics.LowerBoundary.M(int64(lo.Height())))
		s.bound.SetLo(lo)
	}

	err := s.mgr.SetBoundary(s.name, s.bound.Boundary)
	if err == nil {
		return nil
	}

	s.bound.Boundary = prev
	return err
}

// SetBoundary updates the boundary of the segment
func (s *Segment) SetBoundary(ctx context.Context, hi, lo *common.LinkedTipSet) error {
	s.bound.Lock()
	defer s.bound.Unlock()

	// TODO: reset, cleanup and other stuffs
	return s.updateBoundary(ctx, hi, lo)
}

// ReadBoundary returns the boundary of the segment
func (s *Segment) ReadBoundary() Boundary {
	s.bound.RLock()
	b := s.bound.Boundary
	s.bound.RUnlock()
	return b
}

// Name returns the name of the segment
func (s *Segment) Name() string {
	return s.name
}

// ReadDB returns the read only db instance of the segment
func (s *Segment) ReadDB() common.DocumentDB {
	return s.rdb
}

// SaveFinalHeight saves final height
func (s *Segment) SaveFinalHeight(ctx context.Context, hi *common.LinkedTipSet) error {
	elog := log.With("finalHeight", hi.Height())
	elog.Info("save final height")
	res := extract.NewRes(0, 0)
	docs := make([][]common.Document, 1)

	doc, err := model.NewFinalHeight(hi)
	if err != nil {
		return err
	}

	res.Docs = append(res.Docs, doc)
	docs[0] = res.Docs
	if err := s.insertMany(ctx, elog, docs); err != nil {
		return fmt.Errorf("SaveFinalHeight err: %w", err)
	}

	return nil
}

func (s *Segment) GetFinalHeight(ctx context.Context) (abi.ChainEpoch, error) {
	log.Info("get final height")

	findOpts := make([]*options.FindOptions, 0)
	findOpts = append(findOpts, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}), options.Find().SetLimit(-1))
	cursor, err := s.rdb.Find(ctx, "FinalHeight", bson.D{}, findOpts...)
	if err != nil {
		return 0, err
	}

	type FinalHeightRes struct {
		Epoch abi.ChainEpoch `bson:"_id"`
		Cids  []string
	}

	var results []FinalHeightRes
	if err = cursor.All(context.TODO(), &results); err != nil {
		return 0, err
	}

	if len(results) != 1 {
		return 0, fmt.Errorf("unexpected length of results: %v", len(results))
	}

	return results[0].Epoch, nil
}

func (s *Segment) SaveStateFinalHeight(ctx context.Context, hi *common.LinkedTipSet) error {
	elog := log.With("stateFinalHeight", hi.Height())
	elog.Info("save state final height")
	res := extract.NewRes(0, 0)
	docs := make([][]common.Document, 1)

	doc, err := model.NewStateFinalHeight(hi)
	if err != nil {
		return err
	}

	res.Docs = append(res.Docs, doc)
	docs[0] = res.Docs
	if err := s.insertMany(ctx, elog, docs); err != nil {
		return fmt.Errorf("SaveFinalHeight err: %w", err)
	}

	return nil
}

func (s *Segment) GetStateFinalHeight(ctx context.Context) (abi.ChainEpoch, error) {
	log.Info("get final height")

	findOpts := make([]*options.FindOptions, 0)
	findOpts = append(findOpts, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}), options.Find().SetLimit(-1))
	cursor, err := s.rdb.Find(ctx, "StateFinalHeight", bson.D{}, findOpts...)
	if err != nil {
		return 0, err
	}

	type StateFinalHeightRes struct {
		Epoch abi.ChainEpoch `bson:"_id"`
		Cids  []string
	}

	var results []StateFinalHeightRes
	if err = cursor.All(context.TODO(), &results); err != nil {
		return 0, err
	}

	if len(results) != 1 {
		return 0, fmt.Errorf("unexpected length of results: %v", len(results))
	}

	return results[0].Epoch, nil
}

func (s *Segment) GetLatestDealID(ctx context.Context) (int64, error) {
	log.Info("get latest dealID")

	findOpts := make([]*options.FindOptions, 0)
	findOpts = append(findOpts, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}), options.Find().SetLimit(-1))
	cursor, err := s.rdb.Find(ctx, "NewDealProposal", bson.D{}, findOpts...)
	if err != nil {
		return 0, err
	}

	type dealProposalRes struct {
		ID int64 `bson:"_id"`
	}

	var results []dealProposalRes
	if err = cursor.All(context.TODO(), &results); err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return -1, nil
	}

	log.Infof("get latest dealID: %v", results[0].ID)

	return results[0].ID, nil
}
