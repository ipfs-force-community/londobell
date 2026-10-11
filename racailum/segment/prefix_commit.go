package segment

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

// prefixCommitDeps 是 commitPrefix 的依赖集合，全部经回调注入，让「连续前缀提交」这段
// 控制流可以在不构造真实 *Segment（不连链、不连 mongo）的前提下被单测覆盖——单测据此
// 证明「Aggregate 与 SaveFinalHeight 只吃前缀、绝不跨洞」这条正确性红线。
type prefixCommitDeps struct {
	// aggregate 执行跨高度聚合。★ 红线：必须只吃前缀——漏掉这一处就等于跨洞发布派生表
	// （接口 200 + 无错日志 + 错误数值），正是本设计要避免的形态。
	aggregate func(ctx context.Context, prefix []*common.LinkedTipSet) error
	// saveFinalHeight 前移对外发布水位。★ 红线：必须只推前缀末，绝不推到整批末。
	saveFinalHeight func(ctx context.Context, head *common.LinkedTipSet) error
	// saveStateHeight 与 saveFinalHeight 二选一（OnlyExtractState 时用），同样只推前缀末。
	saveStateHeight func(ctx context.Context, head *common.LinkedTipSet) error
	// updateBoundary 段边界前移（lo = nil 表示不动 lo）。
	updateBoundary func(ctx context.Context, head *common.LinkedTipSet) error

	onlyExtractState bool
	saveStateEnabled bool
}

// prefixLenBeforeFirstHole 返回 tss 中「首个阻断面之前」的连续高度数（即安全可发布的前缀长度）。
//
// 阻断面的定位：取 fails（= 已登记的洞 ∪ 失败 part）里 FromEpoch 最小者的起点作为首个阻断面的
// 起点（见 firstBlockingBoundary）；前缀 = tss 中高度**严格小于**该起点的高度数。tss 按高度
// 单调递增，所以这等价于「阻断面之前那一段连续 tipset 的数量」。fails 为空 ⇒ 无阻断 ⇒
// 返回 len(tss)（整批都是前缀）。
//
// 为什么以高度（唯一业务键）而不是「失败 part 在 tss 里的下标」为准：partRunner 记录的
// 是 part 的 [FromEpoch,ToEpoch]，part 边界与 tss 下标只是间接关系，用高度最稳。
//
// 纯函数：只读入参、无副作用，便于表驱动单测。与 firstBlockingBoundary 同源（都从同一份
// fails 取最早起点），避免两处判据各写一份而漂移。
func prefixLenBeforeFirstHole(tss []*common.LinkedTipSet, fails []PartFailure) int {
	b, ok := firstBlockingBoundary(fails)
	if !ok {
		return len(tss)
	}

	n := 0
	for _, ts := range tss {
		if ts.Height() < b.FromEpoch {
			n++
		}
	}
	return n
}

// PrefixLenBeforeFirstHole 是 prefixLenBeforeFirstHole 的导出包装。
// 存在的理由：ra.go 的离线抽取路径（RaCailum.WalkExtract）也必须遵守同一条
// 「绝不跨洞」判据（见 docs/cn/batch_failure_semantics.md §8.3），
// 而它不在 segment 包内，需要有个导出的入口复用同一份逻辑——
// 复制一份判据到那边迟早会漂移。
func PrefixLenBeforeFirstHole(tss []*common.LinkedTipSet, fails []PartFailure) int {
	return prefixLenBeforeFirstHole(tss, fails)
}

// BlockingBoundary 描述本轮最早的「阻断面」：本轮**不能越过**、下一轮必须重做的起始高度。
// 它的来源只有两类（见 PartFailure.Kind）：已登记的洞（BlockedByHole）与失败 part
// （BlockedByPersistFailure）。
type BlockingBoundary struct {
	FromEpoch abi.ChainEpoch // 阻断面起始高度（part / 洞的起始高度）
	ToEpoch   abi.ChainEpoch // 阻断面结束高度
	Kind      string         // BlockedByHole | BlockedByPersistFailure
}

// firstBlockingBoundary 是「求第一个阻断面」的纯函数：从失败清单（= 已登记的洞 ∪ 失败 part）
// 里取起始高度最小的一个。清单为空 ⇒ ok=false（无阻断，整批可发布）。
//
// 不假设入参有序：逐个取高度最小者。Kind 为空的旧式字面量按 BlockedByHole 处理
// （既有只按 Class/区间构造 PartFailure 的调用方语义不变）。
func firstBlockingBoundary(fails []PartFailure) (BlockingBoundary, bool) {
	if len(fails) == 0 {
		return BlockingBoundary{}, false
	}

	best := fails[0]
	for _, f := range fails[1:] {
		if f.FromEpoch < best.FromEpoch {
			best = f
		}
	}

	kind := best.Kind
	if kind == "" {
		kind = BlockedByHole
	}
	return BlockingBoundary{FromEpoch: best.FromEpoch, ToEpoch: best.ToEpoch, Kind: kind}, true
}

// FirstBlockingBoundary 是 firstBlockingBoundary 的导出包装，便于跨包复用与测试
// （口径同 PrefixLenBeforeFirstHole）。
func FirstBlockingBoundary(fails []PartFailure) (BlockingBoundary, bool) {
	return firstBlockingBoundary(fails)
}

// commitPrefix 承载「连续前缀提交」这段控制流：算前缀 → **只把前缀**交给依赖。
//
// ★ 正确性红线（docs/cn/batch_failure_semantics.md §8.3）：Aggregate 与
// SaveFinalHeight/SaveStateFinalHeight 都必须只吃前缀；updateBoundary 同理只推前缀末。
//
// 阻断面语义：前缀 = 第一个阻断面（已登记的洞 ∪ 失败 part）之前那一段连续 tipset。
//
//   - 无阻断时（fails 为空）prefixLen == len(tss) ⇒ prefix 就是整批，行为与改动前完全一致；
//   - 阻断面在批中 ⇒ 只发布 [批首, 阻断面-1]，水位只前移到前缀末，后续即使有已成功的 part
//     也绝不跨越阻断面发布（**本轮不算致命失败**，算部分成功）；
//   - 阻断面就在批首（prefixLen == 0）⇒ 不发布、水位不动，同样**不算致命失败**：下一轮会从
//     同一位置续跑。
//
// 返回非 nil error 只表示「发布本身落地失败」（Aggregate / SaveFinalHeight / updateBoundary
// 报错），那是真正的致命错误，由上层整批弃。
func commitPrefix(ctx context.Context, tss []*common.LinkedTipSet, fails []PartFailure, deps prefixCommitDeps) error {
	prefixLen := prefixLenBeforeFirstHole(tss, fails)

	if prefixLen == 0 {
		// 阻断面就在批首：没有任何「连续、可安全发布」的部分。不发布、水位不动，
		// 但**不是致命失败**——下一轮从同一位置续跑（这是本次改动相对「整批作废」的关键区别）。
		logBlockedBeforePublish(tss, fails)
		return nil
	}

	prefix := tss[:prefixLen]
	head := prefix[prefixLen-1]

	if err := deps.aggregate(ctx, prefix); err != nil {
		return err
	}

	if deps.onlyExtractState {
		if deps.saveStateEnabled {
			if err := deps.saveStateHeight(ctx, head); err != nil {
				return err
			}
		}
	} else {
		if err := deps.saveFinalHeight(ctx, head); err != nil {
			return err
		}
	}

	if err := deps.updateBoundary(ctx, head); err != nil {
		return err
	}

	logPublishedPrefix(tss, head, fails)

	return nil
}

// logPublishedPrefix 记录一次成功的前缀发布（可观测性，便于主网排障）：
//   - INFO：range / publishedTo / blockedBy / blockedAt；
//   - 若本轮被阻断面截断（存在阻断面），再补一条 WARN 说明「本轮被阻断面截断，水位停在 X」。
func logPublishedPrefix(tss []*common.LinkedTipSet, head *common.LinkedTipSet, fails []PartFailure) {
	b, blocked := firstBlockingBoundary(fails)
	if !blocked {
		log.Infow("published contiguous prefix to batch end",
			"range", common.FormatTipSetEpochRange(tss),
			"publishedTo", int64(head.Height()),
			"blockedBy", "none")
		return
	}

	log.Infow("published contiguous prefix truncated by blocking boundary",
		"range", common.FormatTipSetEpochRange(tss),
		"publishedTo", int64(head.Height()),
		"blockedBy", b.Kind, "blockedAt", int64(b.FromEpoch))

	log.Warnw("round truncated by blocking boundary, watermark stopped",
		"watermark", int64(head.Height()),
		"blockedBy", b.Kind, "blockedAt", int64(b.FromEpoch),
		"range", common.FormatTipSetEpochRange(tss))
}

// logBlockedBeforePublish 记录「阻断面就在批首、本轮什么都没发布、水位不动」的情形。
func logBlockedBeforePublish(tss []*common.LinkedTipSet, fails []PartFailure) {
	if len(tss) == 0 {
		return
	}
	b, blocked := firstBlockingBoundary(fails)
	if !blocked {
		return
	}
	log.Warnw("batch head blocked by blocking boundary, nothing published, watermark unchanged",
		"range", common.FormatTipSetEpochRange(tss),
		"blockedBy", b.Kind, "blockedAt", int64(b.FromEpoch))
}
