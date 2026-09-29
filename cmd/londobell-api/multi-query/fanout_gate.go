package multiquery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
)

// ---------------------------------------------------------------------------
// 背景：聚合器对 mongo 分片的出站查询是「无界物化 + 无全局并发上限」。
//
//   - 每个多库扇出函数（MultiTraversalQuery / MultiPagingQuery / MultiRangeQuery /
//     MultiUnionQuery / MultiTraversalQueryByCid ...）对每个命中库发一个 goroutine，
//     库里再 `cur.All()` 把整个结果集一次性读进内存；
//   - 单个 HTTP 请求因此会同时挂起「库数」个结果缓冲区；N 个并发请求就是 N×库数，
//     没有任何全局上限；
//   - 冷盘（EBS 改型 / HDD 冷存储）变慢时，单个查询占用令牌的时间变长，
//     请求到达速率大于完成速率 ⇒ 在飞查询数无界增长 ⇒ 分配速率把 RSS 顶到 cgroup 上限
//     （生产实测：活堆 598MB、RSS 8.37GB，6 分钟内被 OOM 杀 4 次）。
//
// 这个文件提供两个机制，都只做「有界」不做「截断」：
//
//	A. 全局出站分片查询闸门（shardQueryGate，默认开启）：进程内所有分片聚合/探测查询
//	   共享一份令牌，令牌在「结果物化完成」后才释放，于是「全进程在飞的结果缓冲区数」
//	   有上限。拿不到令牌时**有界等待**，等不到就返回 *FanoutSaturatedError* ——
//	   显式错误，绝不返回空结果、绝不部分返回。
//	B. 扇出家族端点的请求级闸门（fanoutRequestGate，默认关闭）：对 trace_for_message /
//	   blocks_for_message / hash_by_messagecid / child_transfers_for_message 这类重活入口
//	   限制并发请求数，超限直接 503 快速失败（而不是先做一半工作再失败）。默认关闭，
//	   作为「宁可少收请求也不做一半」的运维开关。
//
// 正常负载下两个闸门都不应成为瓶颈（默认 limit 高于线上观测的在飞查询数），
// 因此正常路径的返回值与改前逐字节一致（单测覆盖）。
// ---------------------------------------------------------------------------

const (
	// DefaultShardQueryConcurrency 是全局出站分片查询并发上限的默认值。
	//
	// 推导（不是拍脑袋）：线上观测的 mongo 连接数在正常负载下是 110–212，而每个在飞的
	// 分片聚合在物化期间占用一个连接，因此在飞查询数 ≈ 连接数 ≈ 212。默认取 256
	// （2 的幂，约 20% 余量）⇒ 正常负载永不触发等待；冷盘变慢时在飞数被钉在 256，
	// 相对事故期（RSS 8.37GB / 基线 1.07GB ≈ 8× 膨胀 ⇒ 在飞约 1600+）是 ~6× 收敛，
	// 峰值内存回落到基线附近，远低于 8.4GB 的 cgroup 硬上限。
	DefaultShardQueryConcurrency = 256

	// DefaultShardQueryWait 是获取令牌的有界等待时长：等不到就显式报错（不截断）。
	// 取 5s：正常负载下令牌随手可得（不等待）；冷盘变慢时宁可快速失败（前端已有 500/503
	// 语义）也不要把请求挂在内存里几十秒。
	DefaultShardQueryWait = 5 * time.Second

	// ShardQueryConcurrencyEnv 环境变量：>0 = 并发上限；<=0 = 关闭闸门（回退旧行为）。
	ShardQueryConcurrencyEnv = "LONDOBELL_SHARD_QUERY_CONCURRENCY"
	// ShardQueryWaitSecondsEnv 环境变量：>0 = 等待秒数；0 = 用默认；<0 = 无限等待（直到 ctx 取消）。
	ShardQueryWaitSecondsEnv = "LONDOBELL_SHARD_QUERY_WAIT_SECONDS"
	// FanoutRequestConcurrencyEnv 环境变量：扇出家族请求级并发上限（<=0 = 关闭）。
	FanoutRequestConcurrencyEnv = "LONDOBELL_FANOUT_REQUEST_CONCURRENCY"
)

// FanoutSaturatedError 表示「拿不到出站分片查询令牌」。它是一个显式错误：
// 调用方必须把它当失败上报，绝不能当成「这个库没有数据」而返回空/部分结果。
type FanoutSaturatedError struct {
	Limit int
	Wait  time.Duration
}

func (e *FanoutSaturatedError) Error() string {
	return fmt.Sprintf("shard query concurrency budget exhausted (limit=%d, waited=%v); "+
		"refusing to run the query instead of returning incomplete data", e.Limit, e.Wait)
}

// IsFanoutSaturated 判断 err 是否是（或包装了）FanoutSaturatedError。
func IsFanoutSaturated(err error) bool {
	var target *FanoutSaturatedError
	return errors.As(err, &target)
}

// RequestGateSaturatedError 表示扇出家族端点的请求级并发闸已满（B 方案）。
type RequestGateSaturatedError struct {
	Limit int
}

func (e *RequestGateSaturatedError) Error() string {
	return fmt.Sprintf("fanout request gate saturated (limit=%d)", e.Limit)
}

// ---------------------------------------------------------------------------
// A. 全局出站分片查询闸门
// ---------------------------------------------------------------------------

// shardQueryGate 是一个可热重配的信号量。
//
// 重配（改 limit/wait）只是把 ch 换成新的 channel；在飞的持有者把 release 闭包
// 绑在「自己拿到的那个 channel」上，因此重配不会让它们 release 到错误的 channel。
// 代价：重配瞬间在飞数可能短暂超过新 limit（旧 token 还没还），这是一次性的、
// 方向安全（只会更松不会更紧），可接受。
type shardQueryGate struct {
	mu    sync.RWMutex
	limit int
	wait  time.Duration
	ch    chan struct{}

	inflight  int64
	peak      int64
	acquired  int64
	saturated int64
	lastWarn  int64
}

var shardGate = newShardQueryGate(DefaultShardQueryConcurrency, DefaultShardQueryWait)

func newShardQueryGate(limit int, wait time.Duration) *shardQueryGate {
	g := &shardQueryGate{}
	g.reconfigure(limit, wait)
	return g
}

func (g *shardQueryGate) reconfigure(limit int, wait time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.limit = limit
	g.wait = wait
	if limit > 0 {
		g.ch = make(chan struct{}, limit)
	} else {
		g.ch = nil // limit<=0 = 关闭闸门
	}
}

// acquire 取一个令牌，返回释放函数。失败时返回错误（绝不静默跳过）。
//
//   - limit<=0：闸门关闭，直通（但仍计入在飞计数，便于观测）；
//   - wait<=0：一直等到 ctx 取消（无限等待）；
//   - wait>0：有界等待，超时返回 *FanoutSaturatedError。
func (g *shardQueryGate) acquire(ctx context.Context) (func(), error) {
	g.mu.RLock()
	ch, limit, wait := g.ch, g.limit, g.wait
	g.mu.RUnlock()

	// onAcquired 统一记录在飞/峰值计数。ch==nil（闸门关闭）时不做 channel 操作。
	onAcquired := func(ch chan struct{}) func() {
		atomic.AddInt64(&g.acquired, 1)
		cur := atomic.AddInt64(&g.inflight, 1)
		for {
			peak := atomic.LoadInt64(&g.peak)
			if cur <= peak || atomic.CompareAndSwapInt64(&g.peak, peak, cur) {
				break
			}
		}

		var once sync.Once
		return func() {
			once.Do(func() {
				atomic.AddInt64(&g.inflight, -1)
				if ch != nil {
					<-ch
				}
			})
		}
	}

	if limit <= 0 || ch == nil {
		return onAcquired(nil), nil
	}

	if wait <= 0 {
		select {
		case ch <- struct{}{}:
			return onAcquired(ch), nil
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire shard query slot: %w", ctx.Err())
		}
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case ch <- struct{}{}:
		return onAcquired(ch), nil
	case <-ctx.Done():
		return nil, fmt.Errorf("acquire shard query slot: %w", ctx.Err())
	case <-timer.C:
		atomic.AddInt64(&g.saturated, 1)
		g.warnSaturated(limit, wait)
		return nil, &FanoutSaturatedError{Limit: limit, Wait: wait}
	}
}

// warnSaturated 限流打点：最多每秒一条，避免饱和时把日志刷爆。
func (g *shardQueryGate) warnSaturated(limit int, wait time.Duration) {
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&g.lastWarn)
	if now-last < int64(time.Second) {
		return
	}
	if !atomic.CompareAndSwapInt64(&g.lastWarn, last, now) {
		return
	}

	log.Warnf("shard query gate saturated: limit=%d, waited=%v, inflight=%d, peak=%d, saturated_total=%d",
		limit, wait, atomic.LoadInt64(&g.inflight), atomic.LoadInt64(&g.peak), atomic.LoadInt64(&g.saturated))
}

// ShardGateStats 是闸门计数快照（用于日志/排查）。
type ShardGateStats struct {
	Limit     int   `json:"limit"`
	WaitMS    int64 `json:"wait_ms"`
	InFlight  int64 `json:"inflight"`
	Peak      int64 `json:"peak"`
	Acquired  int64 `json:"acquired"`
	Saturated int64 `json:"saturated"`
}

func ShardGateSnapshot() ShardGateStats {
	g := shardGate
	g.mu.RLock()
	limit, wait := g.limit, g.wait
	g.mu.RUnlock()

	return ShardGateStats{
		Limit:     limit,
		WaitMS:    wait.Milliseconds(),
		InFlight:  atomic.LoadInt64(&g.inflight),
		Peak:      atomic.LoadInt64(&g.peak),
		Acquired:  atomic.LoadInt64(&g.acquired),
		Saturated: atomic.LoadInt64(&g.saturated),
	}
}

// SetShardGateLimit 直接设置闸门参数（测试 / 运维热调钩子）。
// limit<=0 = 关闭；wait<=0 = 无限等待直到 ctx 取消。
func SetShardGateLimit(limit int, wait time.Duration) {
	shardGate.reconfigure(limit, wait)
}

// resetShardGateStats 清零闸门计数（只在确认没有在飞查询时调用，例如测试里两轮
// 测量之间）。令牌语义不受影响。
func resetShardGateStats() {
	g := shardGate
	atomic.StoreInt64(&g.inflight, 0)
	atomic.StoreInt64(&g.peak, 0)
	atomic.StoreInt64(&g.acquired, 0)
	atomic.StoreInt64(&g.saturated, 0)
}

// ResolveShardGatePolicy 计算生效策略：环境变量 > 配置文件 > 内置默认。
// 每次调用现算，所以改配置文件（30s 巡检热生效）与改环境变量（需重启）都能生效。
func ResolveShardGatePolicy(cfg common.Config) (limit int, wait time.Duration) {
	limit = DefaultShardQueryConcurrency
	switch {
	case cfg.ShardQueryConcurrency > 0:
		limit = cfg.ShardQueryConcurrency
	case cfg.ShardQueryConcurrency < 0:
		limit = 0 // 关闭
	}

	wait = DefaultShardQueryWait
	switch {
	case cfg.ShardQueryWaitSeconds > 0:
		wait = time.Duration(cfg.ShardQueryWaitSeconds) * time.Second
	case cfg.ShardQueryWaitSeconds < 0:
		wait = 0 // 无限等待
	}

	if v := strings.TrimSpace(os.Getenv(ShardQueryConcurrencyEnv)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n <= 0 {
				limit = 0
			} else {
				limit = n
			}
		}
	}

	if v := strings.TrimSpace(os.Getenv(ShardQueryWaitSecondsEnv)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			wait = time.Duration(n) * time.Second // n<0 = 无限等待，n=0 = 立即失败
		}
	}

	return limit, wait
}

// ApplyShardGatePolicy 从配置计算并应用闸门策略。进程启动与配置重载时调用。
func ApplyShardGatePolicy(cfg common.Config) {
	limit, wait := ResolveShardGatePolicy(cfg)
	shardGate.reconfigure(limit, wait)
	log.Infof("shard query gate policy applied: limit=%d, wait=%v", limit, wait)
}

// ---------------------------------------------------------------------------
// 出站分片查询执行：闸门 + 物化
// ---------------------------------------------------------------------------

// shardQueryRunner 在单个集合上执行一次出站聚合并全量物化结果。
// 抽成函数类型 + 包级变量，便于单测注入假实现（不连真库）。
type shardQueryRunner func(ctx context.Context, col *mongo.Collection, pipe interface{}, allowDiskUse bool) ([]bson.M, error)

// shardQueryFunc 是当前使用的分片查询实现。测试可临时替换（务必用 defer 还原）。
var shardQueryFunc shardQueryRunner = mongoShardQuery

// mongoShardQuery 是真实实现：聚合 + cur.All 全量物化。
// AllowDiskUse 的口径与原实现一致（BlockMessage 需要）。
func mongoShardQuery(ctx context.Context, col *mongo.Collection, pipe interface{}, allowDiskUse bool) ([]bson.M, error) {
	var (
		cur *mongo.Cursor
		err error
	)
	if allowDiskUse {
		cur, err = col.Aggregate(ctx, pipe, options.Aggregate().SetAllowDiskUse(true))
	} else {
		cur, err = col.Aggregate(ctx, pipe)
	}
	if err != nil {
		return nil, err
	}

	var res []bson.M
	if err := cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// withShardSlot 在全局闸门内执行一次分片出站操作。
// fn 里通常包含 `col.Aggregate(...)` + `cur.All(ctx, dst)`：物化必须发生在持有令牌期间，
// 这样「全进程在飞的结果缓冲区数」才有上限。
// 拿不到令牌时返回错误，绝不静默跳过。
func withShardSlot(ctx context.Context, fn func() error) error {
	release, err := shardGate.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	return fn()
}

// runShardQuery 在全局闸门内执行一次分片聚合查询（结果类型固定为 []bson.M）。
//
// 关键点：物化（cur.All 把整个结果集读进内存）发生在**持有令牌期间**，
// 所以全进程「在飞的结果缓冲区」数量被钉在闸门上限内。慢分片只会让单个令牌被占用更久
// （后续请求有界等待），不会让并发数无界增长。
//
// 走 shardQueryFunc 间接层，便于单测注入假实现（不连真库）。
// 拿不到令牌时返回错误，绝不返回空结果。
func runShardQuery(ctx context.Context, col *mongo.Collection, pipe interface{}, allowDiskUse bool) ([]bson.M, error) {
	var res []bson.M
	err := withShardSlot(ctx, func() error {
		var err error
		res, err = shardQueryFunc(ctx, col, pipe, allowDiskUse)
		return err
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// runShardOp 在全局闸门内执行一次任意的分片出站操作（例如覆盖索引探测的 find）。
// 探测虽然便宜，但同样占用一个 mongo 连接，一并纳入预算才能把连接数钉住。
func runShardOp(ctx context.Context, fn func(ctx context.Context) (bool, error)) (bool, error) {
	release, err := shardGate.acquire(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	return fn(ctx)
}

// ---------------------------------------------------------------------------
// B. 扇出家族端点的请求级闸门（默认关闭）
// ---------------------------------------------------------------------------

// fanoutRequestGate 限制「重活扇出端点」的并发请求数：满了就直接拒绝（快速失败），
// 而不是先做一半工作再把内存拖爆。默认 limit=0（关闭），只做运维开关。
type fanoutRequestGate struct {
	mu    sync.RWMutex
	limit int
	ch    chan struct{}

	inflight int64
	rejected int64
	lastWarn int64
}

var fanoutGate = &fanoutRequestGate{}

func (g *fanoutRequestGate) reconfigure(limit int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.limit = limit
	if limit > 0 {
		g.ch = make(chan struct{}, limit)
	} else {
		g.ch = nil
	}
}

// AcquireFanoutRequest 非阻塞地取一个请求级令牌。返回错误 = 应立刻 503（不做任何工作）。
func AcquireFanoutRequest() (func(), error) {
	g := fanoutGate

	g.mu.RLock()
	ch, limit := g.ch, g.limit
	g.mu.RUnlock()

	if limit <= 0 || ch == nil {
		return func() {}, nil
	}

	select {
	case ch <- struct{}{}:
		atomic.AddInt64(&g.inflight, 1)
		var once sync.Once
		return func() {
			once.Do(func() {
				atomic.AddInt64(&g.inflight, -1)
				<-ch
			})
		}, nil
	default:
		atomic.AddInt64(&g.rejected, 1)
		g.warnRejected(limit)
		return nil, &RequestGateSaturatedError{Limit: limit}
	}
}

func (g *fanoutRequestGate) warnRejected(limit int) {
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&g.lastWarn)
	if now-last < int64(time.Second) {
		return
	}
	if !atomic.CompareAndSwapInt64(&g.lastWarn, last, now) {
		return
	}

	log.Warnf("fanout request gate rejected request: limit=%d, inflight=%d, rejected_total=%d",
		limit, atomic.LoadInt64(&g.inflight), atomic.LoadInt64(&g.rejected))
}

// FanoutRequestGateStats 请求级闸门计数快照。
type FanoutRequestGateStats struct {
	Limit    int   `json:"limit"`
	InFlight int64 `json:"inflight"`
	Rejected int64 `json:"rejected"`
}

func FanoutRequestGateSnapshot() FanoutRequestGateStats {
	g := fanoutGate
	g.mu.RLock()
	limit := g.limit
	g.mu.RUnlock()

	return FanoutRequestGateStats{
		Limit:    limit,
		InFlight: atomic.LoadInt64(&g.inflight),
		Rejected: atomic.LoadInt64(&g.rejected),
	}
}

// ResolveFanoutRequestGatePolicy 计算请求级闸门上限：环境变量 > 配置文件 > 默认（关闭）。
func ResolveFanoutRequestGatePolicy(cfg common.Config) int {
	limit := 0
	if cfg.FanoutRequestConcurrency > 0 {
		limit = cfg.FanoutRequestConcurrency
	}

	if v := strings.TrimSpace(os.Getenv(FanoutRequestConcurrencyEnv)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n <= 0 {
				limit = 0
			} else {
				limit = n
			}
		}
	}

	return limit
}

// ApplyFanoutRequestGatePolicy 应用请求级闸门策略（进程启动 / 配置重载）。
func ApplyFanoutRequestGatePolicy(cfg common.Config) {
	limit := ResolveFanoutRequestGatePolicy(cfg)
	fanoutGate.reconfigure(limit)
	log.Infof("fanout request gate policy applied: limit=%d", limit)
}

// SetFanoutRequestGateLimit 直接设置请求级闸门上限（测试 / 运维钩子）。<=0 = 关闭。
func SetFanoutRequestGateLimit(limit int) {
	fanoutGate.reconfigure(limit)
}
