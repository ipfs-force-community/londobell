package segment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

// 线上事故（2026-10-08 主网）：单个 tipset 的抽取内部走节点 RPC，go-jsonrpc 在连接
// 异常时会卡在等 nil channel 上永不返回；原实现只给作业 cancel-only 的 context，
// 于是一次卡死 = 整批永久停摆、游标不动、进程却"看着还活着"（supervisord 只看进程在不在）。
// 这组测试锁住修复后的行为：作业永不返回时必须被放弃、重试、最终报错，绝不无限等待。

func fastRetry(t *testing.T) {
	t.Helper()
	old := jobRetryBackoff
	jobRetryBackoff = 5 * time.Millisecond
	t.Cleanup(func() { jobRetryBackoff = old })
}

// 永不返回且不响应 ctx 的作业（复现 go-jsonrpc 卡死）必须被超时放弃并重试。
func TestExtractJobWithTimeout_HangIsAbandonedAndRetried(t *testing.T) {
	fastRetry(t)
	s := &Segment{}
	timeout := 40 * time.Millisecond
	attempts := 2

	var calls int
	start := time.Now()
	_, err := s.extractJobWithTimeout(context.Background(), timeout, attempts, zap.NewNop().Sugar(), "hang",
		func(_ context.Context) (*extract.Res, error) {
			calls++
			select {} // 永不返回，也不理会 ctx
		})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("作业卡死时必须报错，实际返回 nil")
	}
	if calls != attempts {
		t.Fatalf("期望尝试 %d 次，实际 %d 次", attempts, calls)
	}
	if limit := timeout*time.Duration(attempts) + jobRetryBackoff*time.Duration(attempts-1) + 2*time.Second; elapsed > limit {
		t.Fatalf("耗时 %s 超出上限 %s（说明没有按超时放弃）", elapsed, limit)
	}
	var te *extractJobTimeoutErr
	if !errors.As(err, &te) {
		t.Fatalf("错误链里应含超时类型，实际 %v", err)
	}
}

// 第一次失败、第二次成功：重试要真的再跑一次，且返回成功那次的结果。
func TestExtractJobWithTimeout_RetrySucceeds(t *testing.T) {
	fastRetry(t)
	s := &Segment{}

	var calls int
	res, err := s.extractJobWithTimeout(context.Background(), time.Second, 3, zap.NewNop().Sugar(), "flaky",
		func(_ context.Context) (*extract.Res, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("boom")
			}
			r := extract.NewRes(4, 0)
			r.Docs = append(r.Docs, nil)
			return r, nil
		})
	if err != nil {
		t.Fatalf("第二次应成功，实际报错 %v", err)
	}
	if calls != 2 {
		t.Fatalf("期望尝试 2 次，实际 %d 次", calls)
	}
	if res == nil || len(res.Docs) != 1 {
		t.Fatalf("应返回成功那次的结果，实际 %+v", res)
	}
}

// 上层已取消时不得继续重试，且应返回上层错误（避免取消后还在打节点）。
func TestExtractJobWithTimeout_ParentCanceled(t *testing.T) {
	fastRetry(t)
	s := &Segment{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int
	_, err := s.extractJobWithTimeout(ctx, time.Second, 3, zap.NewNop().Sugar(), "canceled",
		func(_ context.Context) (*extract.Res, error) {
			calls++
			return nil, errors.New("should not be called")
		})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("期望 context.Canceled，实际 %v", err)
	}
	if calls != 0 {
		t.Fatalf("上层已取消时不应执行作业，实际执行 %d 次", calls)
	}
}

// 异步落库等待同样必须有上限：Mongo 写入挂住时不能永久等待。
func TestWaitAsyncPersist_Timeout(t *testing.T) {
	var g multierror.Group
	g.Go(func() error {
		select {} // 永不完成
	})

	if err := waitAsyncPersist(&g, 40*time.Millisecond); err == nil {
		t.Fatal("等待挂住时必须报错，实际返回 nil")
	}
}

func TestWaitAsyncPersist_OK(t *testing.T) {
	var g multierror.Group
	g.Go(func() error { return nil })

	if err := waitAsyncPersist(&g, time.Second); err != nil {
		t.Fatalf("正常完成不应报错，实际 %v", err)
	}
}
