package util

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRunBoundedLimitsConcurrency 固化「任务数量不决定并发数」这条约束。
// 1000 项时同时在跑的任务不得超过 BoundedConcurrency，且每一项都被执行一次。
// 这是 2026-10-24 OOM（每项一个 goroutine）的回归护栏。
func TestRunBoundedLimitsConcurrency(t *testing.T) {
	const n = 1000

	var (
		mu      sync.Mutex
		cur     int
		peak    int
		visited = make([]bool, n)
	)

	err := RunBounded(context.Background(), n, func(ctx context.Context, i int) error {
		mu.Lock()
		cur++
		if cur > peak {
			peak = cur
		}
		mu.Unlock()

		time.Sleep(time.Millisecond)

		mu.Lock()
		cur--
		visited[i] = true
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("非预期错误: %v", err)
	}
	if peak > BoundedConcurrency {
		t.Fatalf("并发度 %d 超过上限 %d", peak, BoundedConcurrency)
	}
	for i, ok := range visited {
		if !ok {
			t.Fatalf("第 %d 项未被执行", i)
		}
	}
}

// TestRunBoundedJoinsErrors 出错时仍执行完全部任务，并把错误合并返回。
func TestRunBoundedJoinsErrors(t *testing.T) {
	const n = 60

	var done int64
	err := RunBounded(context.Background(), n, func(ctx context.Context, i int) error {
		atomic.AddInt64(&done, 1)
		if i == 3 || i == 9 {
			return errors.New("boom")
		}
		return nil
	})
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
	if got := atomic.LoadInt64(&done); got != n {
		t.Fatalf("只执行了 %d 项，期望 %d 项全部执行", got, n)
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		if got := len(joined.Unwrap()); got != 2 {
			t.Fatalf("合并后的错误数 = %d，期望 2", got)
		}
	}
}

// TestRunBoundedEmpty 边界：0 项与负数都不应启动 worker。
func TestRunBoundedEmpty(t *testing.T) {
	for _, n := range []int{0, -5} {
		err := RunBounded(context.Background(), n, func(context.Context, int) error {
			return errors.New("不应被调用")
		})
		if err != nil {
			t.Fatalf("n=%d 时应返回 nil，实际 %v", n, err)
		}
	}
}
