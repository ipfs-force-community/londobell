package multiquery

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRunSegmentQueriesBoundedConcurrency 固化「段数不影响并发度」这条约束。
// 用 1000 个段跑：同时在执行的段数不得超过 pagingConcurrency，且每个段都被执行一次。
// 这条用例正是 2026-10-24 OOM 事故的回归护栏——原实现每段起一个 goroutine，
// 1000 段就会有 1000 个 goroutine 同时存在（其中 984 个只是在排队）。
func TestRunSegmentQueriesBoundedConcurrency(t *testing.T) {
	const n = 1000

	var (
		mu      sync.Mutex
		cur     int
		peak    int
		visited = make([]bool, n)
	)

	err := runSegmentQueries(context.Background(), n, func(ctx context.Context, i int) error {
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

	if peak > pagingConcurrency {
		t.Fatalf("并发度 %d 超过上限 %d（段数不应决定协程数）", peak, pagingConcurrency)
	}
	for i, ok := range visited {
		if !ok {
			t.Fatalf("段 %d 未被执行", i)
		}
	}
}

// TestRunSegmentQueriesErrorPropagates 出错时仍然执行完全部段，并把错误返回给调用方
// （与 multierror.Group 的语义一致：宁可失败，也不返回不完整数据）。
func TestRunSegmentQueriesErrorPropagates(t *testing.T) {
	const n = 50

	var done int64
	err := runSegmentQueries(context.Background(), n, func(ctx context.Context, i int) error {
		atomic.AddInt64(&done, 1)
		if i == 7 {
			return errors.New("boom")
		}
		return nil
	})
	if err == nil {
		t.Fatal("期望返回错误，实际返回 nil")
	}
	if got := atomic.LoadInt64(&done); got != n {
		t.Fatalf("只执行了 %d 段，期望 %d 段全部执行", got, n)
	}
}

// TestRunSegmentQueriesEmpty 边界：0 段时不启动 worker、不执行任务。
func TestRunSegmentQueriesEmpty(t *testing.T) {
	err := runSegmentQueries(context.Background(), 0, func(context.Context, int) error {
		return errors.New("不应被调用")
	})
	if err != nil {
		t.Fatalf("空集合应返回 nil，实际 %v", err)
	}
}
