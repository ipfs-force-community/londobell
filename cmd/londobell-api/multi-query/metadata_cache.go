package multiquery

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// 元数据扇出缓存（只缓存「各库高度区间 / 各库总条数」，绝不缓存业务数据）
//
// refresh() 每被调用一次，就向全部冷库 + 正式库并发发一轮查询、再单独跑一次 tmp 库，
// 目的只是拿到 CountUtil —— 各库负责的高度区间、各库总条数、分段 state 这类元数据。
// 聚合器对外约 20 个统计端点每个请求都要先做这件事，扇出的固定成本 ≈ 库数 × 单库随机读延迟，
// 在慢盘（HDD 冷存储）上实测同一端点 86ms → 352ms。高度跨度大多只有 1 个 epoch，
// 每请求重新扇出一遍纯属浪费。
//
// 这里给 refresh() 加一层「短 TTL 缓存 + 单飞（singleflight）合并」：
//   - 入缓存的只有 refresh() 的返回值（CountUtil），即元数据；业务数据（消息/区块/事件行）
//     依然每个请求真查，不经过这里；
//   - TTL 可配置、可关闭（配置文件 / 环境变量），关闭后完全回退到「每次请求都扇出一遍」；
//   - 未命中时同一 key 的并发请求只打出一轮扇出，其余等在这一轮结果上（20 个端点同时打进来
//     也只扇出一轮）；
//   - key 里带 curEpoch（height = (now-BaseTime)/30，每 30s 变一次），因此命中只发生在
//     同一个 epoch 内，跨 epoch 不复用，不会少算最新高度；
//   - 新高度/新库的可见延迟 ≤ TTL，配置重载与 formal state 刷新时会主动失效（=立即可见）。

const (
	// DefaultMetaCacheTTL refresh() 元数据缓存的默认 TTL。
	// 取 30s（与 epoch 长度同量级）：key 里带 curEpoch，每 30s 天然换桶，
	// TTL 只需兜住「同一个 epoch 内库高度区间/条数的漂移」。
	DefaultMetaCacheTTL = 30 * time.Second

	// MetaCacheTTLEnv 环境变量（秒）：>0 覆盖配置文件里的 TTL；<=0 关闭缓存。
	MetaCacheTTLEnv = "LONDOBELL_METADATA_CACHE_TTL"
	// MetaCacheDisableEnv 环境变量：1/true 时关闭缓存（下线开关）。
	MetaCacheDisableEnv = "LONDOBELL_METADATA_CACHE_DISABLED"

	// metaCacheLoadTimeout 共享扇出的兜底上限：不跟任何单个请求的 ctx 取消，
	// 但也不能无限期挂着。给足余量（单请求基线 20-35s，队列尾部 60s+，
	// mongo socket 超时 10 分钟），只用来兜住真正卡死的扇出。
	metaCacheLoadTimeout = 3 * time.Minute

	// defaultMetaCacheMaxEntries 缓存条目上限。key 含用户输入（actor 地址 / methodName），
	// 必须封顶，否则不同 actor 的请求会无限堆积 key。
	defaultMetaCacheMaxEntries = 256
)

// metaCachePolicy 计算当前生效的缓存策略：环境变量 → 配置文件 → 内置默认（开启，30s）。
// 每次调用现算，所以改配置/（重启进程后）改环境变量都能直接生效，无需重新接线。
func metaCachePolicy(cfg common.Config) (enabled bool, ttl time.Duration) {
	enabled = !cfg.DisableMetaCache
	ttl = DefaultMetaCacheTTL
	if cfg.MetaCacheTTLSeconds > 0 {
		ttl = time.Duration(cfg.MetaCacheTTLSeconds) * time.Second
	}

	if v := strings.TrimSpace(os.Getenv(MetaCacheTTLEnv)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			if n <= 0 {
				enabled = false
			} else {
				enabled, ttl = true, time.Duration(n)*time.Second
			}
		}
	}

	if v := strings.TrimSpace(os.Getenv(MetaCacheDisableEnv)); v == "1" || strings.EqualFold(v, "true") {
		enabled = false
	}

	return enabled, ttl
}

// metaCacheKey 组装缓存键。四个维度缺一不可：
//   - flavor：哪个 refresh 收集器（决定 CountUtil 里哪些字段被填充，也与 f 一一对应）；
//   - actorID / methodName：同一收集器在不同筛选条件下的结果不同（也可能为空串）；
//   - curEpoch：tmp 库的分段结束高度是 curEpoch+1，跨 epoch 复用会让查询范围少算最新高度。
func metaCacheKey(flavor, actorID, methodName string, curEpoch abi.ChainEpoch) string {
	return fmt.Sprintf("%s|actor=%s|method=%s|epoch=%d", flavor, actorID, methodName, int64(curEpoch))
}

// MetaCacheStats 元数据缓存计数快照，用于把命中率打进日志（聚合器没有 prometheus exporter）。
type MetaCacheStats struct {
	Hits              int64 `json:"hits"`
	Misses            int64 `json:"misses"`
	SingleflightWaits int64 `json:"singleflight_waits"`
	Fetches           int64 `json:"fetches"`  // 真正打出的扇出轮数
	Errors            int64 `json:"errors"`   // 扇出失败的轮数
	Bypasses          int64 `json:"bypasses"` // 缓存关闭时的直通次数（旧行为）
	Expired           int64 `json:"expired"`  // 惰性淘汰
	Evicted           int64 `json:"evicted"`  // 触顶淘汰
	Invalidations     int64 `json:"invalidations"`
	Entries           int   `json:"entries"`
	Inflight          int   `json:"inflight"`
}

// Requests 返回参与命中率计算的查询次数（缓存关闭时的直通不计入）。
func (s MetaCacheStats) Requests() int64 {
	return s.Hits + s.Misses
}

// HitRate 命中率，无查询时为 0。
func (s MetaCacheStats) HitRate() float64 {
	total := s.Requests()
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

type metaCacheValue struct {
	utils     []CountUtil
	expiresAt time.Time
}

// metaCacheCall 一轮在飞扇出：等待者通过 done 拿到结果（写入发生在 close 之前）。
type metaCacheCall struct {
	done  chan struct{}
	utils []CountUtil
	err   error
}

// metaCache 并发安全：entries/inflight/gen 由 mu 保护（锁只覆盖 map 读写的临界区，
// 扇出本身在锁外执行）；计数用 atomic，读 Stats 不需要抢锁。
type metaCache struct {
	mu         sync.Mutex
	entries    map[string]metaCacheValue
	inflight   map[string]*metaCacheCall
	gen        uint64 // invalidation 代数：在飞扇出若跨代则不入缓存
	maxEntries int
	now        func() time.Time // 可注入，便于测 TTL

	hits, misses, sfWaits, fetches, errs, bypasses, expired, evicted, invalidations int64
}

// metadataCache 是 refresh() 使用的全局实例。测试里可直接替换（同包）。
var metadataCache = newMetaCache()

func newMetaCache() *metaCache {
	return &metaCache{
		entries:    make(map[string]metaCacheValue),
		inflight:   make(map[string]*metaCacheCall),
		maxEntries: defaultMetaCacheMaxEntries,
		now:        time.Now,
	}
}

// MetaCacheStatsSnapshot 返回全局缓存的计数快照，便于排查/验证命中率。
func MetaCacheStatsSnapshot() MetaCacheStats {
	return metadataCache.Stats()
}

// GetOrLoad 取缓存；未命中则由本调用者打出一轮扇出（load），同一 key 的其它并发请求
// 合并到这一轮上。enabled=false 或 ttl<=0 时完全直通（= 旧行为）。
func (c *metaCache) GetOrLoad(ctx context.Context, key string, enabled bool, ttl time.Duration,
	load func(context.Context) ([]CountUtil, error)) ([]CountUtil, error) {
	if load == nil {
		return nil, fmt.Errorf("nil load func for metadata cache key %v", key)
	}

	if !enabled || ttl <= 0 {
		atomic.AddInt64(&c.bypasses, 1)
		return load(ctx)
	}

	now := c.now()

	c.mu.Lock()
	if val, ok := c.entries[key]; ok {
		if now.Before(val.expiresAt) {
			utils := cloneCountUtils(val.utils)
			c.mu.Unlock()
			atomic.AddInt64(&c.hits, 1)
			log.Debugf("metadata cache hit: key=%v, expiresIn=%v", key, val.expiresAt.Sub(now).String())
			return utils, nil
		}
		// 过期即删（惰性淘汰），继续走未命中路径
		delete(c.entries, key)
		atomic.AddInt64(&c.expired, 1)
	}

	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		atomic.AddInt64(&c.sfWaits, 1)
		select {
		case <-call.done:
			return cloneCountUtils(call.utils), call.err
		case <-ctx.Done():
			// 等待者自己的请求被取消：立即返回，不拖住它
			return nil, ctx.Err()
		}
	}

	call := &metaCacheCall{done: make(chan struct{})}
	c.inflight[key] = call
	gen := c.gen
	c.mu.Unlock()

	atomic.AddInt64(&c.misses, 1)

	// 共享扇出不跟着某一个请求的 ctx 取消：这一轮结果要给等待中的其它请求复用，
	// 不能让单个客户端断连把所有人的结果一起变成 error。ctx 里的取值（TableKey 等）保留。
	loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), metaCacheLoadTimeout)
	defer cancel()

	start := c.now()
	utils, err := load(loadCtx)
	elapsed := c.now().Sub(start)

	c.finishCall(key, call, utils, err, gen, ttl)
	c.logFetch(key, ttl, elapsed, utils, err)

	// 直接返回刚扇出的那份：入缓存的是它的 clone，调用方就地改不到缓存。
	return utils, err
}

func (c *metaCache) finishCall(key string, call *metaCacheCall, utils []CountUtil, err error, gen uint64, ttl time.Duration) {
	c.mu.Lock()
	// 存 clone：调用方拿到的那份会被上层就地回填
	//（例如 controller/aggregators/blockheader_messages_by_methodname.go
	// 会写 countUtils[i].BlockHeaderMethodStates），不能让它污染缓存里的副本。
	if err == nil && c.gen == gen {
		c.storeLocked(key, cloneCountUtils(utils), c.now().Add(ttl))
	}
	delete(c.inflight, key)
	c.mu.Unlock()

	call.utils = utils
	call.err = err
	close(call.done)

	if err != nil {
		atomic.AddInt64(&c.errs, 1)
	} else {
		atomic.AddInt64(&c.fetches, 1)
	}
}

func (c *metaCache) storeLocked(key string, utils []CountUtil, expiresAt time.Time) {
	limit := c.maxEntries
	if limit < 1 {
		limit = 1
	}

	if len(c.entries) >= limit {
		c.pruneLocked()
	}

	// 仍然触顶：淘汰最早过期的那个（key 可能来自用户输入，必须封顶）
	for len(c.entries) >= limit {
		var (
			victim     string
			victimExp  time.Time
			firstEntry = true
		)
		for k, v := range c.entries {
			if firstEntry || v.expiresAt.Before(victimExp) {
				victim, victimExp, firstEntry = k, v.expiresAt, false
			}
		}

		delete(c.entries, victim)
		atomic.AddInt64(&c.evicted, 1)
	}

	c.entries[key] = metaCacheValue{utils: utils, expiresAt: expiresAt}
}

func (c *metaCache) pruneLocked() {
	now := c.now()
	for k, v := range c.entries {
		if !now.Before(v.expiresAt) {
			delete(c.entries, k)
			atomic.AddInt64(&c.expired, 1)
		}
	}
}

// Invalidate 丢弃全部缓存，并让在飞的扇出结果不再入缓存（gen 自增）。
// 配置重载（新冷库接入 / 库列表变化）与 formal state 刷新后调用：
// 这些变化影响 CountUtil 的区间，主动失效可让可见延迟从「≤TTL」变成「立即」。
func (c *metaCache) Invalidate(reason string) {
	c.mu.Lock()
	c.gen++
	dropped := len(c.entries)
	c.entries = make(map[string]metaCacheValue)
	c.mu.Unlock()

	atomic.AddInt64(&c.invalidations, 1)
	if dropped > 0 {
		log.Infof("metadata cache invalidated: reason=%v, dropped=%d entries", reason, dropped)
	}
}

// Stats 返回计数快照。
func (c *metaCache) Stats() MetaCacheStats {
	c.mu.Lock()
	entries, inflight := len(c.entries), len(c.inflight)
	c.mu.Unlock()

	return MetaCacheStats{
		Hits:              atomic.LoadInt64(&c.hits),
		Misses:            atomic.LoadInt64(&c.misses),
		SingleflightWaits: atomic.LoadInt64(&c.sfWaits),
		Fetches:           atomic.LoadInt64(&c.fetches),
		Errors:            atomic.LoadInt64(&c.errs),
		Bypasses:          atomic.LoadInt64(&c.bypasses),
		Expired:           atomic.LoadInt64(&c.expired),
		Evicted:           atomic.LoadInt64(&c.evicted),
		Invalidations:     atomic.LoadInt64(&c.invalidations),
		Entries:           entries,
		Inflight:          inflight,
	}
}

// logFetch 每真正打出一轮扇出就记一行，带上累计计数 —— 上线后 grep 这一行即可算命中率。
func (c *metaCache) logFetch(key string, ttl, elapsed time.Duration, utils []CountUtil, err error) {
	s := c.Stats()

	if err != nil {
		log.Warnf("metadata cache fetch failed: key=%v, elapsed=%v, err=%v, hits=%d, misses=%d, "+
			"singleflight_waits=%d, fetches=%d, errors=%d, entries=%d, hit_rate=%.4f",
			key, elapsed, err, s.Hits, s.Misses, s.SingleflightWaits, s.Fetches, s.Errors, s.Entries, s.HitRate())
		return
	}

	log.Infof("metadata cache fetch: key=%v, ttl=%v, elapsed=%v, count_utils=%d, hits=%d, misses=%d, "+
		"singleflight_waits=%d, fetches=%d, errors=%d, bypasses=%d, entries=%d, hit_rate=%.4f",
		key, ttl, elapsed, len(utils), s.Hits, s.Misses, s.SingleflightWaits, s.Fetches, s.Errors,
		s.Bypasses, s.Entries, s.HitRate())
}

// cloneCountUtils 复制一份 CountUtil 切片给调用方。
// 两个原因：(1) 上层会就地回填/改写 CountUtil 字段（见 finishCall 注释），共享底层数组会污染缓存；
// (2) 同一份元数据会被多个 goroutine 同时读，复制后可各自按需修改（例如排序分段 state）而不互相干扰。
// Cols 里的 mongo 句柄是只读共享（与原行为一致），不复制。
func cloneCountUtils(utils []CountUtil) []CountUtil {
	if utils == nil {
		return nil
	}

	out := make([]CountUtil, len(utils))
	copy(out, utils)
	for i := range out {
		out[i].BlockStates = cloneSegmentStates(out[i].BlockStates)
		out[i].BlockMethodStates = cloneSegmentStates(out[i].BlockMethodStates)
	}

	return out
}

func cloneSegmentStates(states []smodel.SegmentState) []smodel.SegmentState {
	if states == nil {
		return nil
	}

	out := make([]smodel.SegmentState, len(states))
	copy(out, states)

	return out
}
