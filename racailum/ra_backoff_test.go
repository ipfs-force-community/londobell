package racailum

import (
	"context"
	"testing"
	"time"
)

// 轮间退避（§8.6）：失败一轮后按指数退避等待，成功一轮立即复位。
// 这组测试锁住两件事：①退避曲线 5m→10m→20m→上限 30m 且封顶不回退；
// ②成功复位由调用方（roundBackoffState）承担，复位后下一次失败必须回到 initial。
// 两者都不睡眠：nextBackoff 是纯函数，roundBackoffState 只维护计数。

func TestNextBackoff(t *testing.T) {
	const (
		initial = 5 * time.Minute
		max     = 30 * time.Minute
	)

	cases := []struct {
		name        string
		consecutive int
		initial     time.Duration
		max         time.Duration
		want        time.Duration
	}{
		// 递增 + 封顶：5m → 10m → 20m → 30m → 30m
		{"第 1 次失败取 initial", 1, initial, max, 5 * time.Minute},
		{"第 2 次失败翻倍到 10m", 2, initial, max, 10 * time.Minute},
		{"第 3 次失败翻倍到 20m", 3, initial, max, 20 * time.Minute},
		{"第 4 次失败撞上限取 30m", 4, initial, max, 30 * time.Minute},
		{"第 5 次失败仍封顶 30m", 5, initial, max, 30 * time.Minute},
		{"第 10 次失败仍封顶 30m", 10, initial, max, 30 * time.Minute},
		{"第 1000 次失败仍封顶 30m（不回退/不溢出）", 1000, initial, max, 30 * time.Minute},

		// 边界：非正 consecutive 视为第一次失败
		{"consecutive=0 视为第 1 次", 0, initial, max, 5 * time.Minute},
		{"consecutive<0 视为第 1 次", -7, initial, max, 5 * time.Minute},

		// 边界：initial == max 时立即封顶
		{"initial 即为上限", 5, max, max, 30 * time.Minute},
		{"上下限相等时第 1 次即封顶", 1, max, max, 30 * time.Minute},

		// 边界：配置异常（非正）退化为「不退避」，与改动前等价，绝不卡死循环
		{"initial 非正退化为不退避", 3, 0, max, 0},
		{"max 非正退化为不退避", 3, initial, 0, 0},
		{"initial 为负退化为不退避", 3, -time.Minute, max, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := nextBackoff(c.consecutive, c.initial, c.max); got != c.want {
				t.Fatalf("nextBackoff(%d, %s, %s) = %s, want %s",
					c.consecutive, c.initial, c.max, got, c.want)
			}
		})
	}
}

// 覆盖调用方复位逻辑：连续失败走完 5m→10m→20m→30m→30m 后，
// 只要有「成功一轮」，下一次失败必须立即回到 initial（而不是接着封顶值）。
func TestRoundBackoffState_ResetOnSuccess(t *testing.T) {
	b := roundBackoffState{
		initial: 5 * time.Minute,
		max:     30 * time.Minute,
	}

	want := []time.Duration{
		5 * time.Minute,
		10 * time.Minute,
		20 * time.Minute,
		30 * time.Minute,
		30 * time.Minute, // 已封顶，再多失败也不变
	}
	for i, w := range want {
		if got := b.fail(); got != w {
			t.Fatalf("第 %d 次连续失败退避 = %s, want %s", i+1, got, w)
		}
	}

	// 成功一轮 ⇒ 复位。
	b.success()
	if b.consecutive != 0 {
		t.Fatalf("成功一轮后 consecutive 应复位为 0，实际 %d", b.consecutive)
	}
	if got := b.fail(); got != 5*time.Minute {
		t.Fatalf("复位后的第一次失败退避 = %s, want %s", got, 5*time.Minute)
	}
}

// backoffBeforeNextRound 是 Run 循环里「失败轮 / 被阻断面截断轮」的共用退避入口。
// 覆盖三件事：①到期后返回 true（继续循环）；②ctx 取消 ⇒ false（退出循环）；
// ③doneCh 关闭 ⇒ false。后两者用长退避时长确保 select 命中的是取消而非 time.After（确定性）。
func TestBackoffBeforeNextRound_WaitsThenContinues(t *testing.T) {
	b := roundBackoffState{initial: 2 * time.Millisecond, max: 2 * time.Millisecond}

	start := time.Now()
	if !backoffBeforeNextRound(context.Background(), make(chan struct{}), &b, "test") {
		t.Fatal("未被取消时应返回 true 以继续循环")
	}
	if elapsed := time.Since(start); elapsed < 1*time.Millisecond {
		t.Fatalf("应真的等待了退避时长，实际仅 %s", elapsed)
	}
	if b.consecutive != 1 {
		t.Fatalf("一次退避后 consecutive 应为 1，实际 %d", b.consecutive)
	}
}

func TestBackoffBeforeNextRound_CtxCanceledStopsLoop(t *testing.T) {
	b := roundBackoffState{initial: time.Hour, max: time.Hour}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if backoffBeforeNextRound(ctx, make(chan struct{}), &b, "test") {
		t.Fatal("ctx 已取消时应返回 false 退出循环")
	}
}

func TestBackoffBeforeNextRound_DoneChStopsLoop(t *testing.T) {
	b := roundBackoffState{initial: time.Hour, max: time.Hour}
	done := make(chan struct{})
	close(done)

	if backoffBeforeNextRound(context.Background(), done, &b, "test") {
		t.Fatal("doneCh 关闭时应返回 false 退出循环")
	}
}

// 配置异常（退避时长非正）⇒ 不退避、立即返回 true，绝不卡死循环（与改动前等价）。
func TestBackoffBeforeNextRound_NoBackoffWhenConfiguredZero(t *testing.T) {
	b := roundBackoffState{initial: 0, max: 0}

	if !backoffBeforeNextRound(context.Background(), make(chan struct{}), &b, "test") {
		t.Fatal("退避时长为 0 时应立即返回 true")
	}
}
