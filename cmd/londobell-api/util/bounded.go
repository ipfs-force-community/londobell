package util

import (
	"context"
	"errors"
	"sync"
)

// BoundedConcurrency 是「按项起任务」类场景的默认并发上限。
// 取值与 multi-query 的段级并发一致（16），便于运维用同一个心智模型看待扇出。
const BoundedConcurrency = 16

// RunBounded 用固定大小的 worker 池执行 n 个任务，fn 的入参是任务下标。
//
// 为什么需要它：2026-10-24 主网聚合器一天被 cgroup OOM 杀 8 次，两个真凶是同一类写法——
// 「每处理一项就起一个 goroutine」，而项数由外部输入决定：
//   - multi-query.MultiPagingQuery：段数由 index/limit 决定（limit=0 ⇒ 全量段），
//     事故时堆出 897,539 个 goroutine，其中 896,523 个只是在排队；
//   - aggregators.GetMpool：mpool 条数由链上待打包消息量决定，每条一个 goroutine
//     且各自打一次 lotus RPC（GetMethodName），mpool 一涨就是几千并发 RPC。
//
// 用固定 worker 池后，goroutine 数量与「外部输入有多大」解耦。
//
// 语义与 multierror.Group 对齐：所有任务都会被尝试执行（不是第一个错误就中断），
// 返回的错误用 errors.Join 合并；n <= 0 时直接返回 nil。
func RunBounded(ctx context.Context, n int, fn func(ctx context.Context, i int) error) error {
	if n <= 0 {
		return nil
	}

	workers := BoundedConcurrency
	if n < workers {
		workers = n
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	jobs := make(chan int)
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				if err := fn(ctx, i); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			}
		}()
	}

	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
