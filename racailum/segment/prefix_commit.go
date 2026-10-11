package segment

import (
	"context"
	"fmt"

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

// prefixLenBeforeFirstHole 返回 tss 中「首个洞之前」的连续高度数（即安全可发布的前缀长度）。
//
// 洞的定位：取 fails 里 FromEpoch 最小者的起点作为首个洞的起点；前缀 = tss 中高度
// **严格小于**该起点的高度数。tss 按高度单调递增，所以这等价于「洞之前那一段连续 tipset
// 的数量」。fails 为空 ⇒ 无洞 ⇒ 返回 len(tss)（整批都是前缀）。
//
// 为什么以高度（唯一业务键）而不是「失败 part 在 tss 里的下标」为准：partRunner 记录的
// 是 part 的 [FromEpoch,ToEpoch]，part 边界与 tss 下标只是间接关系，用高度最稳。
//
// 纯函数：只读入参、无副作用，便于表驱动单测。
func prefixLenBeforeFirstHole(tss []*common.LinkedTipSet, fails []PartFailure) int {
	if len(fails) == 0 {
		return len(tss)
	}

	// 最早就出现的洞的起点（也叫「首个洞」）。
	first := fails[0].FromEpoch
	for _, f := range fails[1:] {
		if f.FromEpoch < first {
			first = f.FromEpoch
		}
	}

	n := 0
	for _, ts := range tss {
		if ts.Height() < first {
			n++
		}
	}
	return n
}

// commitPrefix 承载「连续前缀提交」这段控制流：算前缀 → **只把前缀**交给依赖。
//
// ★ 正确性红线（docs/cn/batch_failure_semantics.md §8.3）：Aggregate 与
// SaveFinalHeight/SaveStateFinalHeight 都必须只吃前缀；updateBoundary 同理只推前缀末。
//
// 兼容性红线：无洞时（fails 为空）prefixLen == len(tss) ⇒ prefix 就是整批，行为与改动前
// 完全一致。洞在批首（prefixLen == 0）时不发布任何部分、返回 error（保持不动水位）。
func commitPrefix(ctx context.Context, tss []*common.LinkedTipSet, fails []PartFailure, deps prefixCommitDeps) error {
	prefixLen := prefixLenBeforeFirstHole(tss, fails)
	if prefixLen == 0 {
		// 洞就在批首：没有任何「连续、可安全发布」的部分，保持不动水位（与现状一致）。
		return fmt.Errorf("batch head blocked by hole(s): %s", summarizeFailures(fails))
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

	return nil
}
