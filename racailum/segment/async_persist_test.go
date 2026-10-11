package segment

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// 改动 ②（docs/cn/batch_failure_semantics.md §8.2 注入口 ②）：异步落库失败/超时做
// **per-part 归属**，使其成为阻断面（BlockedByPersistFailure），而不是整批作废。
//
// 主网 2026-10-12 事故形态：`error occurs in async persist ... context deadline exceeded`——
// 旧实现把整批的异步落库错误合并成一个 error 直接整批作废（99.2% 已完成的 part 全丢）。
// 这组测试锁住修复后的归属与截断语义。

// 落库失败必须**恰好**归属到失败的那个 part，成功的 part 不产出阻断面。
func TestAsyncPersistTracker_AttributesFailuresToParts(t *testing.T) {
	var tr asyncPersistTracker
	tr.Go(100, 115, func() error { return nil }) // 成功
	tr.Go(116, 131, func() error {
		return errors.New("insert extracted documents from tipsets: context deadline exceeded")
	}) // 失败
	tr.Go(132, 147, func() error { return nil }) // 成功

	fails := tr.wait(time.Second)
	if len(fails) != 1 {
		t.Fatalf("应恰好归因 1 个失败 part，实际 %d（%+v）", len(fails), fails)
	}
	if fails[0].FromEpoch != 116 || fails[0].ToEpoch != 131 {
		t.Fatalf("失败应归属到 part [116,131]，实际 [%d,%d]", fails[0].FromEpoch, fails[0].ToEpoch)
	}
}

// 超时只归因「仍未完成」的作业：已成功的 part 不该被扣上超时。
func TestAsyncPersistTracker_TimeoutAttributesOnlyPendingJobs(t *testing.T) {
	var tr asyncPersistTracker
	tr.Go(10, 11, func() error { return nil }) // 很快成功
	tr.Go(12, 13, func() error { select {} })  // 卡住、永不返回

	fails := tr.wait(200 * time.Millisecond)
	if len(fails) != 1 {
		t.Fatalf("只应把卡住的 part 归因为超时，实际 %d（%+v）", len(fails), fails)
	}
	if fails[0].FromEpoch != 12 || fails[0].ToEpoch != 13 {
		t.Fatalf("超时应归属到卡住的 part [12,13]，实际 [%d,%d]", fails[0].FromEpoch, fails[0].ToEpoch)
	}
	if !strings.Contains(fails[0].Err.Error(), "async persist wait timeout") {
		t.Fatalf("超时错误正文应含 async persist wait timeout，实际 %q", fails[0].Err.Error())
	}
}

// 落库失败转成阻断面（不写台账、不进洞）：Kind=persist-failure，类别用 classifyErr 复用同一份判据。
func TestPartPersistFailuresToBlocking(t *testing.T) {
	fails := []partPersistFailure{{
		FromEpoch: 116,
		ToEpoch:   131,
		Err:       errors.New("insert extracted documents from tipsets: context deadline exceeded"),
	}}

	gots := partPersistFailuresToBlocking(fails)
	if len(gots) != 1 {
		t.Fatalf("应恰好 1 个阻断面，实际 %d（%+v）", len(gots), gots)
	}
	pf := gots[0]
	if pf.FromEpoch != 116 || pf.ToEpoch != 131 {
		t.Fatalf("阻断面区间 = 落库失败的 part [116,131]，实际 [%d,%d]", pf.FromEpoch, pf.ToEpoch)
	}
	if pf.Kind != BlockedByPersistFailure {
		t.Fatalf("异步落库失败必须标为阻断面（persist-failure），实际 %q", pf.Kind)
	}
	if pf.Class != FailureClassTransport {
		t.Fatalf("context deadline exceeded 应归 transport，实际 %q", pf.Class)
	}
	if _, recordable := classifyErr(errors.New("x: context deadline exceeded")); recordable {
		t.Fatal("transport 绝不能可登记（否则瞬时故障会被固化成永久洞）")
	}
	if pf.ErrDigest == "" {
		t.Fatalf("阻断面应带错误摘要，便于排障")
	}
}

// 端到端语义（纯函数链）：落库失败 part 起于 102 ⇒ 可发布前缀 = [批首,101]，
// 失败之后的已成功 part 绝不发布过去——这正是「异步落库失败不再作废整批」的落点。
func TestPartPersistFailureTruncatesPublishPrefix(t *testing.T) {
	tss := mkTSS(t, 100, 101, 102, 103, 104)

	var tr asyncPersistTracker
	tr.Go(102, 103, func() error {
		return errors.New("insert extracted documents from tipsets: context deadline exceeded")
	})

	blocking, _ := asyncPersistOutcome(false, tr.wait(time.Second))
	if got := prefixLenBeforeFirstHole(tss, blocking); got != 2 {
		t.Fatalf("落库失败 part 起于 102 ⇒ 可发布前缀应为 2（100,101），实际 %d", got)
	}
	if b, ok := firstBlockingBoundary(blocking); !ok || b.FromEpoch != 102 || b.Kind != BlockedByPersistFailure {
		t.Fatalf("首个阻断面应为 (from=102, kind=%s)，实际 (%+v, ok=%v)", BlockedByPersistFailure, b, ok)
	}
}

// 全部成功 ⇒ 零阻断面（与旧实现「此时返回 nil error」一致：行为不变）。
func TestAsyncPersistOutcome_NoFailureIsNoop(t *testing.T) {
	for _, tmp := range []bool{false, true} {
		blocking, fatal := asyncPersistOutcome(tmp, nil)
		if blocking != nil || fatal != nil {
			t.Fatalf("无失败时 tmp=%v 应为 (nil,nil)，实际 (%+v,%v)", tmp, blocking, fatal)
		}
	}
}

// asyncPersistOutcome：主路径 ⇒ 阻断面（nil error）；tmp 路径 ⇒ 保持旧行为整批弃（fail-closed）。
// tmp 路径不走 commitPrefix、会把段上界推到整批末，阻断面在那里无人消费 ⇒ 必须整批弃。
func TestAsyncPersistOutcome_MainVsTmp(t *testing.T) {
	fails := []partPersistFailure{{FromEpoch: 5, ToEpoch: 6, Err: errors.New("context deadline exceeded")}}

	blocking, fatal := asyncPersistOutcome(false, fails)
	if fatal != nil {
		t.Fatalf("主路径不应致命，实际 %v", fatal)
	}
	if len(blocking) != 1 || blocking[0].Kind != BlockedByPersistFailure {
		t.Fatalf("主路径应转成 1 个阻断面，实际 %+v", blocking)
	}

	blocking, fatal = asyncPersistOutcome(true, fails)
	if fatal == nil {
		t.Fatal("tmp 路径必须保持旧行为：异步落库失败整批弃（fail-closed）")
	}
	if blocking != nil {
		t.Fatalf("tmp 路径不应转成阻断面（否则边界会被越过失败 part 推进），实际 %+v", blocking)
	}
	if !strings.Contains(fatal.Error(), "error occurs in async persist") {
		t.Fatalf("tmp 路径错误正文应保持 'error occurs in async persist'，实际 %q", fatal.Error())
	}
	if !strings.Contains(fatal.Error(), "[5-6:") {
		t.Fatalf("tmp 路径错误正文应带失败的 part 区间，实际 %q", fatal.Error())
	}
}

// classifyErr 是「分类规则」的单一事实源：classifyPartFailure 与异步落库归因必须同源。
func TestClassifyErr_MatchesClassifyPartFailure(t *testing.T) {
	part := mkTSS(t, 5, 6)
	errs := []error{
		errors.New("blockstore get: ipld: could not find bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"),
		errors.New("decode message: not minimally encoded"),
		errors.New("insert: context deadline exceeded"),
		errors.New("totally unexpected"),
	}
	for _, e := range errs {
		pf, _ := classifyPartFailure(part, e)
		class, _ := classifyErr(e)
		if pf.Class != class {
			t.Fatalf("classifyErr 与 classifyPartFailure 漂移：err=%q classifyErr=%q classifyPartFailure=%q", e, class, pf.Class)
		}
	}
}
