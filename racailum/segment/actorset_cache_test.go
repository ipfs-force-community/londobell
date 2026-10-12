package segment

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
)

// errTestBuild 是注入用的构建失败错误（验证 build 失败不污染缓存）。
var errTestBuild = errors.New("test actor set build failure")

// 本文件覆盖「按网络版本缓存 actor set」的核心判据与状态机（acquireActorSet /
// actorSetInvalidationOf）。全部不连链、不连 mongo：build 以回调注入，用零值 *actor.Set
// 即可分辨「复用」与「重建」，从而确定性地验证四条失效边界。
//
// 缓存是包级全局（segment.Actorset），跨测试共享 ⇒ 每条用例前后都清干净。

// resetActorSetCache 清空全局缓存，并在用例结束时再清一次（避免污染同包其它测试）。
func resetActorSetCache(t *testing.T) {
	t.Helper()
	ClearActorSet()
	t.Cleanup(ClearActorSet)
}

// TestActorSetInvalidationOf 表驱动覆盖四种失效边界与判定优先级（纯函数，确定性）。
func TestActorSetInvalidationOf(t *testing.T) {
	const within = abi.ChainEpoch(1000)

	cases := []struct {
		name string
		snap actorSetSnapshot
		want network.Version
		now  abi.ChainEpoch
		got  actorSetInvalidation
	}{
		{
			name: "无缓存⇒missing",
			snap: actorSetSnapshot{},
			want: 3, now: within,
			got: actorSetMissing,
		},
		{
			name: "版本一致且在预算与窗口内⇒valid",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: 0, builtAt: within},
			want: 3, now: within + 500,
			got: actorSetValid,
		},
		{
			name: "网络版本变化⇒version-changed",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: 0, builtAt: within},
			want: 4, now: within + 500,
			got: actorSetVersionChanged,
		},
		{
			name: "复用轮数达上界⇒budget-exhausted",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: actorSetReuseBudget, builtAt: within},
			want: 3, now: within + 500,
			got: actorSetBudgetExhausted,
		},
		{
			name: "预算边界内一次⇒valid",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: actorSetReuseBudget - 1, builtAt: within},
			want: 3, now: within + 500,
			got: actorSetValid,
		},
		{
			name: "距构建超 epoch 窗口⇒epoch-window-exceeded",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: 0, builtAt: within},
			want: 3, now: within + actorSetEpochWindow,
			got: actorSetWindowExceeded,
		},
		{
			name: "窗口边界内一 epoch⇒valid",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: 0, builtAt: within},
			want: 3, now: within + actorSetEpochWindow - 1,
			got: actorSetValid,
		},
		{
			name: "版本变化优先于预算与窗口⇒version-changed",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: actorSetReuseBudget, builtAt: within},
			want: 4, now: within + actorSetEpochWindow,
			got: actorSetVersionChanged,
		},
		{
			name: "预算优先于窗口⇒budget-exhausted",
			snap: actorSetSnapshot{exists: true, version: 3, reuses: actorSetReuseBudget, builtAt: within},
			want: 3, now: within + actorSetEpochWindow,
			got: actorSetBudgetExhausted,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := actorSetInvalidationOf(tc.snap, tc.want, tc.now); got != tc.got {
				t.Fatalf("actorSetInvalidationOf 期望 %v，实际 %v", tc.got, got)
			}
		})
	}
}

// TestActorSetInvalidationString 锁住日志里出现的可区分原因串（勿随意改动）。
func TestActorSetInvalidationString(t *testing.T) {
	cases := map[actorSetInvalidation]string{
		actorSetValid:           "valid",
		actorSetMissing:         "missing",
		actorSetVersionChanged:  "version-changed",
		actorSetBudgetExhausted: "budget-exhausted",
		actorSetWindowExceeded:  "epoch-window-exceeded",
	}
	// 反向断言：每个原因串唯一，便于日志 grep。
	seen := map[string]actorSetInvalidation{}
	for reason, want := range cases {
		if got := reason.String(); got != want {
			t.Fatalf("%v.String() = %q，期望 %q", reason, got, want)
		}
		if prev, dup := seen[want]; dup {
			t.Fatalf("原因串 %q 重复：%v 与 %v", want, prev, reason)
		}
		seen[want] = reason
	}
}

// countingBuilder 返回一个可注入的 build 回调，并记录被调用次数。
func countingBuilder(builds *int) func() (*actor.Set, error) {
	return func() (*actor.Set, error) {
		*builds++
		return &actor.Set{}, nil
	}
}

// 同一网络版本内复用（不重建）：连续多次 acquire 必须复用同一实例，build 只调一次。
func TestAcquireActorSet_ReusesWithinSameVersion(t *testing.T) {
	resetActorSetCache(t)

	builds := 0
	build := countingBuilder(&builds)

	first, why1, err := acquireActorSet(3, 1000, build)
	if err != nil {
		t.Fatalf("空缓存首次 acquire 不应报错: %v", err)
	}
	if why1 != actorSetMissing {
		t.Fatalf("空缓存首次 acquire 的原因应为 missing，实际 %v", why1)
	}

	for i, epoch := range []abi.ChainEpoch{1010, 1100, 1500} {
		again, why, err := acquireActorSet(3, epoch, build)
		if err != nil {
			t.Fatalf("同版本内第 %d 次 acquire 不应报错: %v", i+1, err)
		}
		if why != actorSetValid {
			t.Fatalf("同版本内第 %d 次 acquire 应复用（valid），实际 %v", i+1, why)
		}
		if again != first {
			t.Fatalf("同版本内第 %d 次 acquire 应复用同一实例", i+1)
		}
	}
	if builds != 1 {
		t.Fatalf("同一网络版本内只应构建一次，实际构建 %d 次", builds)
	}
}

// 网络版本变化 ⇒ 必须重建（即便状态根/高度相邻）。
func TestAcquireActorSet_RebuildsOnVersionChange(t *testing.T) {
	resetActorSetCache(t)

	builds := 0
	build := countingBuilder(&builds)

	first, _, err := acquireActorSet(3, 1000, build)
	if err != nil {
		t.Fatalf("首次 acquire 不应报错: %v", err)
	}

	second, why, err := acquireActorSet(4, 1000, build)
	if err != nil {
		t.Fatalf("版本变化后 acquire 不应报错: %v", err)
	}
	if why != actorSetVersionChanged {
		t.Fatalf("版本变化应判定 version-changed，实际 %v", why)
	}
	if builds != 2 {
		t.Fatalf("版本变化应触发重建（共 2 次构建），实际构建 %d 次", builds)
	}
	if first == second {
		t.Fatal("版本变化后不应复用旧实例")
	}
	if snap := loadActorSet(); !snap.exists || snap.version != 4 {
		t.Fatalf("缓存版本应更新为 4，实际 %+v", snap)
	}
}

// 走到有界失效窗口 ⇒ 重建：复用预算（按轮数）先到。
func TestAcquireActorSet_RebuildsAtReuseBudget(t *testing.T) {
	resetActorSetCache(t)

	builds := 0
	build := countingBuilder(&builds)

	if _, _, err := acquireActorSet(3, 1000, build); err != nil {
		t.Fatalf("首次构建不应报错: %v", err)
	}

	// 预算内的每一次都应是复用（epoch 固定，不触碰 epoch 窗口）。
	for i := 0; i < actorSetReuseBudget; i++ {
		if _, why, err := acquireActorSet(3, 1000, build); err != nil {
			t.Fatalf("预算内第 %d 次 acquire 不应报错: %v", i+1, err)
		} else if why != actorSetValid {
			t.Fatalf("预算内第 %d 次 acquire 应复用，实际 %v", i+1, why)
		}
	}
	if builds != 1 {
		t.Fatalf("预算未耗尽前不应重建，实际构建 %d 次", builds)
	}

	// 第 actorSetReuseBudget+1 次：额度耗尽 ⇒ 重建。
	_, why, err := acquireActorSet(3, 1000, build)
	if err != nil {
		t.Fatalf("预算耗尽后 acquire 不应报错: %v", err)
	}
	if why != actorSetBudgetExhausted {
		t.Fatalf("预算耗尽应判定 budget-exhausted，实际 %v", why)
	}
	if builds != 2 {
		t.Fatalf("预算耗尽应触发重建（共 2 次构建），实际构建 %d 次", builds)
	}
}

// 走到有界失效窗口 ⇒ 重建：epoch 窗口（按链高推进）先到。
func TestAcquireActorSet_RebuildsAtEpochWindow(t *testing.T) {
	resetActorSetCache(t)

	builds := 0
	build := countingBuilder(&builds)

	const builtAt = abi.ChainEpoch(1000)
	if _, _, err := acquireActorSet(3, builtAt, build); err != nil {
		t.Fatalf("首次构建不应报错: %v", err)
	}

	// 窗口内一 epoch（复用额度远未耗尽）。
	if _, why, err := acquireActorSet(3, builtAt+actorSetEpochWindow-1, build); err != nil {
		t.Fatalf("窗口内 acquire 不应报错: %v", err)
	} else if why != actorSetValid {
		t.Fatalf("窗口内应复用，实际 %v", why)
	}

	// 恰好到达窗口上界 ⇒ 重建。
	_, why, err := acquireActorSet(3, builtAt+actorSetEpochWindow, build)
	if err != nil {
		t.Fatalf("超窗口 acquire 不应报错: %v", err)
	}
	if why != actorSetWindowExceeded {
		t.Fatalf("超窗口应判定 epoch-window-exceeded，实际 %v", why)
	}
	if builds != 2 {
		t.Fatalf("超窗口应触发重建（共 2 次构建），实际构建 %d 次", builds)
	}
}

// 构建失败不得污染缓存：应原样返回 error，且不写入半成品缓存。
func TestAcquireActorSet_BuildErrorLeavesCacheUntouched(t *testing.T) {
	resetActorSetCache(t)

	boom := func() (*actor.Set, error) {
		return nil, errTestBuild
	}
	if _, _, err := acquireActorSet(3, 1000, boom); err == nil {
		t.Fatal("build 失败时 acquire 应返回 error")
	}
	if snap := loadActorSet(); snap.exists {
		t.Fatalf("build 失败不应写入缓存，实际 %+v", snap)
	}
}

// 并发调用不 data race（由 -race 检出）：全部用同一版本，从而同时压到「同一实例的并发复用
// （Reuses 读-改-写）」与「首次未命中的并发重建」两种竞争，都走同一把 actorsetMu。
func TestAcquireActorSet_ConcurrentNoRace(t *testing.T) {
	resetActorSetCache(t)

	var builds int64
	build := func() (*actor.Set, error) {
		atomic.AddInt64(&builds, 1)
		return &actor.Set{}, nil
	}

	const workers = 64
	results := make([]*actor.Set, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, _, err := acquireActorSet(7, 5000, build)
			if err != nil {
				t.Errorf("并发 acquire 不应报错: %v", err)
				return
			}
			results[i] = s
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&builds); got < 1 {
		t.Fatalf("并发下至少应构建一次，实际构建 %d 次", got)
	}
	for i, s := range results {
		if s == nil {
			t.Fatalf("第 %d 个 goroutine 返回了 nil actor set", i)
		}
	}

	// 不变量：最终缓存里的集合必然来自某次 build 的返回（每个 store 写入的都是一个 build 结果）。
	actorsetMu.RLock()
	final := Actorset
	actorsetMu.RUnlock()
	if final == nil {
		t.Fatal("并发后缓存应非空")
	}
	found := false
	for _, s := range results {
		if s == final.Set {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("最终缓存的集合必须来自某次 build 的返回")
	}
}
