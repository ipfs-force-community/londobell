package multiquery

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Unix(1700000000, 0)} }

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

// testCountUtils 造一份元数据（各库高度区间 + 条数 + 分段 state），marker 用来区分不同轮扇出。
func testCountUtils(marker int64) []CountUtil {
	return []CountUtil{{
		Start: 100,
		End:   200,
		Cols:  common.Collections{},
		BlockStates: []smodel.SegmentState{{
			Dsn:        "dsn",
			StartEpoch: abi.ChainEpoch(100),
			EndEpoch:   abi.ChainEpoch(200),
			Count:      marker,
		}},
	}}
}

func mustLoad(t *testing.T, c *metaCache, key string, enabled bool, ttl time.Duration, load func(context.Context) ([]CountUtil, error)) []CountUtil {
	t.Helper()

	utils, err := c.GetOrLoad(context.Background(), key, enabled, ttl, load)
	if err != nil {
		t.Fatalf("GetOrLoad(%q) failed: %v", key, err)
	}

	return utils
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timeout waiting for %s", what)
}

// TTL 内命中（不再扇出），TTL 到期后重新扇出。
func TestMetaCacheTTLExpiry(t *testing.T) {
	clock := newFakeClock()
	c := newMetaCache()
	c.now = clock.Now

	const ttl = 30 * time.Second
	var loads int64
	load := func(context.Context) ([]CountUtil, error) {
		return testCountUtils(atomic.AddInt64(&loads, 1)), nil
	}

	first := mustLoad(t, c, "k", true, ttl, load)
	if got := atomic.LoadInt64(&loads); got != 1 {
		t.Fatalf("first call should fan out once, got loads=%d", got)
	}

	// TTL 内：命中，不扇出
	clock.Advance(ttl - time.Second)
	second := mustLoad(t, c, "k", true, ttl, load)
	if got := atomic.LoadInt64(&loads); got != 1 {
		t.Fatalf("within TTL should hit cache, got loads=%d", got)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cache hit returned different value:\nfirst=%+v\nsecond=%+v", first, second)
	}

	// 恰好到期：必须重新扇出
	clock.Advance(time.Second)
	third := mustLoad(t, c, "k", true, ttl, load)
	if got := atomic.LoadInt64(&loads); got != 2 {
		t.Fatalf("at expiry should fan out again, got loads=%d", got)
	}
	if third[0].BlockStates[0].Count != 2 {
		t.Fatalf("expired entry was reused: %+v", third)
	}

	// 过期后一段时间：仍然重新扇出
	clock.Advance(ttl)
	fourth := mustLoad(t, c, "k", true, ttl, load)
	if got := atomic.LoadInt64(&loads); got != 3 {
		t.Fatalf("after expiry should fan out again, got loads=%d", got)
	}
	if fourth[0].BlockStates[0].Count != 3 {
		t.Fatalf("unexpected value: %+v", fourth)
	}

	s := c.Stats()
	if s.Fetches != 3 || s.Misses != 3 || s.Hits != 1 || s.Expired < 2 {
		t.Fatalf("unexpected stats: %+v", s)
	}
	if s.HitRate() != 0.25 {
		t.Fatalf("hit rate should be 1/4, got %v", s.HitRate())
	}
}

// 并发未命中只扇出一轮：其余请求合并（singleflight）到那一轮上。
func TestMetaCacheSingleflightMergesConcurrentMisses(t *testing.T) {
	c := newMetaCache()

	const goroutines = 20
	const ttl = 30 * time.Second

	var loads int64
	loadStarted := make(chan struct{})
	release := make(chan struct{})

	load := func(context.Context) ([]CountUtil, error) {
		if atomic.AddInt64(&loads, 1) == 1 {
			close(loadStarted)
		}
		<-release
		return testCountUtils(42), nil
	}

	type result struct {
		utils []CountUtil
		err   error
	}
	results := make([]result, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			utils, err := c.GetOrLoad(context.Background(), "k", true, ttl, load)
			results[i] = result{utils: utils, err: err}
		}(i)
	}

	<-loadStarted
	// 等其余 goroutine 全部进入「等待同一轮扇出」的分支
	waitFor(t, "all waiters to join the in-flight call", func() bool {
		return c.Stats().SingleflightWaits == goroutines-1
	})
	close(release)
	wg.Wait()

	if got := atomic.LoadInt64(&loads); got != 1 {
		t.Fatalf("concurrent misses must fan out exactly once, got %d", got)
	}

	for i := range results {
		if results[i].err != nil {
			t.Fatalf("goroutine %d failed: %v", i, results[i].err)
		}
		if !reflect.DeepEqual(results[i].utils, testCountUtils(42)) {
			t.Fatalf("goroutine %d got unexpected value: %+v", i, results[i].utils)
		}
	}

	s := c.Stats()
	if s.Fetches != 1 || s.Misses != 1 || s.Hits != 0 || s.SingleflightWaits != goroutines-1 {
		t.Fatalf("unexpected stats: %+v", s)
	}
}

// 关闭开关：每次请求都扇出（完全回退旧行为），且不留缓存条目。
func TestMetaCacheDisabledBypasses(t *testing.T) {
	c := newMetaCache()

	const ttl = 30 * time.Second
	var loads int64
	load := func(context.Context) ([]CountUtil, error) {
		return testCountUtils(atomic.AddInt64(&loads, 1)), nil
	}

	for i := 0; i < 3; i++ {
		mustLoad(t, c, "k", false, ttl, load)
	}
	// enabled=true 但 ttl<=0 同样直通
	for i := 0; i < 2; i++ {
		mustLoad(t, c, "k", true, 0, load)
	}

	if got := atomic.LoadInt64(&loads); got != 5 {
		t.Fatalf("disabled cache must fan out every call, got loads=%d", got)
	}

	s := c.Stats()
	if s.Bypasses != 5 || s.Fetches != 0 || s.Hits != 0 || s.Misses != 0 || s.Entries != 0 {
		t.Fatalf("unexpected stats: %+v", s)
	}
}

// 不同 flavor / actor / method / epoch 必须落在不同的 key 上。
func TestMetaCacheKeyIsolation(t *testing.T) {
	keys := map[string]string{
		metaCacheKey(flavorEpochRange, "a", "m", 1):  "flavor+actor+method+epoch",
		metaCacheKey(flavorEpochRange, "a", "m", 2):  "epoch",
		metaCacheKey(flavorEpochRange, "b", "m", 1):  "actor",
		metaCacheKey(flavorEpochRange, "a", "n", 1):  "method",
		metaCacheKey(flavorTipSetCount, "a", "m", 1): "flavor",
		metaCacheKey(flavorEpochRange, "", "", 1):    "empty actor/method",
		metaCacheKey(flavorActorMsgs, "a", "", 1):    "no method",
		metaCacheKey(flavorActorMsgs, "a", "", 0):    "zero epoch",
	}
	if len(keys) != 8 {
		t.Fatalf("cache key does not distinguish all dimensions: %v", keys)
	}
	if metaCacheKey(flavorEpochRange, "a", "m", 1) != "epoch_range|actor=a|method=m|epoch=1" {
		t.Fatalf("unexpected key format: %v", metaCacheKey(flavorEpochRange, "a", "m", 1))
	}

	// 功能性隔离：不同 key 各自扇出一轮，且互不串值
	c := newMetaCache()
	var loads int64
	load := func(context.Context) ([]CountUtil, error) {
		return testCountUtils(atomic.AddInt64(&loads, 1)), nil
	}

	k1 := metaCacheKey(flavorEpochRange, "actor-1", "", 100)
	k2 := metaCacheKey(flavorEpochRange, "actor-2", "", 100)
	k3 := metaCacheKey(flavorEpochRange, "actor-2", "", 101)

	mustLoad(t, c, k1, true, time.Minute, load) // loads=1
	mustLoad(t, c, k2, true, time.Minute, load) // loads=2
	mustLoad(t, c, k3, true, time.Minute, load) // loads=3
	again := mustLoad(t, c, k1, true, time.Minute, load)

	if got := atomic.LoadInt64(&loads); got != 3 {
		t.Fatalf("distinct keys must not share entries, got loads=%d", got)
	}
	if again[0].BlockStates[0].Count != 1 {
		t.Fatalf("k1 must keep its own value, got %+v", again)
	}
}

// 上层会就地改写 CountUtil（如回填 BlockHeaderMethodStates），缓存副本不能被污染。
func TestMetaCacheReturnedValueIsIsolated(t *testing.T) {
	c := newMetaCache()
	const ttl = time.Minute

	load := func(context.Context) ([]CountUtil, error) {
		return []CountUtil{{
			Start: 1,
			End:   2,
			BlockStates: []smodel.SegmentState{{
				Dsn:   "dsn",
				Count: 7,
			}},
		}}, nil
	}

	first := mustLoad(t, c, "k", true, ttl, load)
	first[0].BlockHeaderMethodStates = 999
	first[0].BlockStates[0].Count = 111
	first[0].End = 4242

	second := mustLoad(t, c, "k", true, ttl, load) // 命中缓存
	if second[0].BlockHeaderMethodStates != 0 || second[0].BlockStates[0].Count != 7 || second[0].End != 2 {
		t.Fatalf("cache entry was poisoned by the caller: %+v", second)
	}
}

// Invalidate：丢弃已有条目；跨代（在飞中失效）的扇出结果不入缓存。
func TestMetaCacheInvalidate(t *testing.T) {
	c := newMetaCache()
	const ttl = time.Minute

	var loads int64
	load := func(context.Context) ([]CountUtil, error) {
		return testCountUtils(atomic.AddInt64(&loads, 1)), nil
	}

	mustLoad(t, c, "k", true, ttl, load)
	if c.Stats().Entries != 1 {
		t.Fatalf("entry should be cached, got %+v", c.Stats())
	}

	c.Invalidate("test")
	if s := c.Stats(); s.Entries != 0 || s.Invalidations != 1 {
		t.Fatalf("invalidate should drop entries, got %+v", s)
	}

	mustLoad(t, c, "k", true, ttl, load)
	if got := atomic.LoadInt64(&loads); got != 2 {
		t.Fatalf("after invalidate the next call must fan out again, got loads=%d", got)
	}

	// 在飞扇出：开始后被 Invalidate，结果不得入缓存
	loadStarted := make(chan struct{})
	release := make(chan struct{})
	blockingLoad := func(context.Context) ([]CountUtil, error) {
		close(loadStarted)
		<-release
		return testCountUtils(100), nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = c.GetOrLoad(context.Background(), "inflight", true, ttl, blockingLoad)
	}()

	<-loadStarted
	c.Invalidate("config reload")
	close(release)
	<-done

	// 在飞结果未入缓存，且没有卡在 inflight 上
	waitFor(t, "inflight to be cleared", func() bool {
		s := c.Stats()
		return s.Inflight == 0 && s.Entries == 0
	})

	mustLoad(t, c, "inflight", true, ttl, load)
	if got := atomic.LoadInt64(&loads); got != 3 {
		t.Fatalf("cross-generation result must not be cached, got loads=%d", got)
	}
}

// 扇出失败不入缓存；等待者拿到同一个错误。
func TestMetaCacheErrorNotCached(t *testing.T) {
	c := newMetaCache()
	const ttl = time.Minute

	boom := errors.New("boom")
	var loads int64

	// 顺序调用：失败两次都不留条目
	for i := 0; i < 2; i++ {
		if _, err := c.GetOrLoad(context.Background(), "k", true, ttl, func(context.Context) ([]CountUtil, error) {
			atomic.AddInt64(&loads, 1)
			return nil, boom
		}); !errors.Is(err, boom) {
			t.Fatalf("expected boom, got %v", err)
		}
	}
	if s := c.Stats(); s.Entries != 0 || s.Errors != 2 || s.Fetches != 0 {
		t.Fatalf("failed fetch must not be cached: %+v", s)
	}

	// 并发：等待者拿到同一个错误，且只扇出一轮
	loadStarted := make(chan struct{})
	release := make(chan struct{})
	failing := func(context.Context) ([]CountUtil, error) {
		close(loadStarted)
		<-release
		return nil, boom
	}

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = c.GetOrLoad(context.Background(), "k2", true, ttl, failing)
		}(i)
	}

	<-loadStarted
	waitFor(t, "waiter to join", func() bool { return c.Stats().SingleflightWaits == 1 })
	close(release)
	wg.Wait()

	for i := range errs {
		if !errors.Is(errs[i], boom) {
			t.Fatalf("goroutine %d expected boom, got %v", i, errs[i])
		}
	}
	if s := c.Stats(); s.Errors != 3 || s.Fetches != 0 || s.Entries != 0 {
		t.Fatalf("unexpected stats: %+v", s)
	}
}

// 条目数封顶：key 含用户输入（actor 地址），不能无限堆积。
func TestMetaCacheMaxEntries(t *testing.T) {
	c := newMetaCache()
	c.maxEntries = 2

	var loads int64
	load := func(context.Context) ([]CountUtil, error) {
		return testCountUtils(atomic.AddInt64(&loads, 1)), nil
	}

	for i := 0; i < 6; i++ {
		mustLoad(t, c, metaCacheKey(flavorActorMsgs, string(rune('a'+i)), "", 1), true, time.Minute, load)
		if got := c.Stats().Entries; got > 2 {
			t.Fatalf("entries must be capped at 2, got %d", got)
		}
	}

	s := c.Stats()
	if s.Evicted == 0 {
		t.Fatalf("expected evictions, got %+v", s)
	}
	if s.Hits != 0 {
		t.Fatalf("no key repeats, so no hits expected: %+v", s)
	}
}

// 关闭/调节 TTL 的开关：环境变量 > 配置文件 > 内置默认（开启，30s）。
func TestMetaCachePolicy(t *testing.T) {
	cases := []struct {
		name        string
		cfg         common.Config
		envTTL      string
		envDisabled string
		wantEnabled bool
		wantTTL     time.Duration
	}{
		{
			name:        "默认：开启 30s",
			cfg:         common.Config{},
			wantEnabled: true,
			wantTTL:     30 * time.Second,
		},
		{
			name:        "配置文件指定 TTL",
			cfg:         common.Config{MetaCacheTTLSeconds: 90},
			wantEnabled: true,
			wantTTL:     90 * time.Second,
		},
		{
			name:        "配置文件关闭",
			cfg:         common.Config{DisableMetaCache: true},
			wantEnabled: false,
			wantTTL:     30 * time.Second,
		},
		{
			name:        "配置文件关闭（TTL 非零也不启用）",
			cfg:         common.Config{DisableMetaCache: true, MetaCacheTTLSeconds: 120},
			wantEnabled: false,
			wantTTL:     120 * time.Second,
		},
		{
			name:        "env 覆盖 TTL",
			cfg:         common.Config{MetaCacheTTLSeconds: 90},
			envTTL:      "10",
			wantEnabled: true,
			wantTTL:     10 * time.Second,
		},
		{
			name:        "env TTL=0 关闭",
			cfg:         common.Config{MetaCacheTTLSeconds: 90},
			envTTL:      "0",
			wantEnabled: false,
			wantTTL:     90 * time.Second,
		},
		{
			name:        "env TTL 非法值忽略",
			cfg:         common.Config{},
			envTTL:      "abc",
			wantEnabled: true,
			wantTTL:     30 * time.Second,
		},
		{
			name:        "env 关闭开关",
			cfg:         common.Config{MetaCacheTTLSeconds: 90},
			envTTL:      "60",
			envDisabled: "1",
			wantEnabled: false,
			wantTTL:     60 * time.Second,
		},
		{
			name:        "env 关闭开关（true）",
			cfg:         common.Config{},
			envDisabled: "true",
			wantEnabled: false,
			wantTTL:     30 * time.Second,
		},
		{
			name:        "env 关闭开关为 0 时不关闭",
			cfg:         common.Config{},
			envDisabled: "0",
			wantEnabled: true,
			wantTTL:     30 * time.Second,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(MetaCacheTTLEnv, tc.envTTL)
			t.Setenv(MetaCacheDisableEnv, tc.envDisabled)

			enabled, ttl := metaCachePolicy(tc.cfg)
			if enabled != tc.wantEnabled || ttl != tc.wantTTL {
				t.Fatalf("metaCachePolicy() = (%v, %v), want (%v, %v)", enabled, ttl, tc.wantEnabled, tc.wantTTL)
			}
		})
	}
}

// 接线测试：Get* 端点走的 refresh() 必须真的经过缓存；关闭时回退到每请求扇出。
func TestRefreshWiringThroughMetadataCache(t *testing.T) {
	orig := metadataCache
	metadataCache = newMetaCache()
	defer func() { metadataCache = orig }()

	// 全部库都 invalid ⇒ refreshUncached 不发任何查询、返回空元数据，但调用链与线上一致
	dbsm := &DataBaseStateManager{DBCfg: common.NewDBCollectionsConfigMgr(common.DefaultConfig())}
	ctx := context.Background()

	utils, err := GetEpochRange(ctx, dbsm, abi.ChainEpoch(1000))
	if err != nil {
		t.Fatalf("GetEpochRange failed: %v", err)
	}
	if len(utils) != 0 {
		t.Fatalf("expected empty metadata, got %+v", utils)
	}

	// 同一 epoch：命中缓存，不再扇出
	if _, err := GetEpochRange(ctx, dbsm, abi.ChainEpoch(1000)); err != nil {
		t.Fatalf("GetEpochRange failed: %v", err)
	}
	if s := MetaCacheStatsSnapshot(); s.Fetches != 1 || s.Hits != 1 {
		t.Fatalf("same epoch should hit the cache: %+v", s)
	}

	// 新 epoch：换 key，重新扇出一次
	if _, err := GetEpochRange(ctx, dbsm, abi.ChainEpoch(1001)); err != nil {
		t.Fatalf("GetEpochRange failed: %v", err)
	}
	if s := MetaCacheStatsSnapshot(); s.Fetches != 2 {
		t.Fatalf("new epoch should fan out again: %+v", s)
	}

	// 不同端点（flavor）不共用条目
	if _, err := GetTotalCountForTipSets(ctx, dbsm, abi.ChainEpoch(1000)); err != nil {
		t.Fatalf("GetTotalCountForTipSets failed: %v", err)
	}
	if s := MetaCacheStatsSnapshot(); s.Fetches != 3 {
		t.Fatalf("different flavor should fan out again: %+v", s)
	}

	// 关闭开关：回退到每请求扇出
	t.Setenv(MetaCacheTTLEnv, "0")
	for i := 0; i < 2; i++ {
		if _, err := GetEpochRange(ctx, dbsm, abi.ChainEpoch(1000)); err != nil {
			t.Fatalf("GetEpochRange failed: %v", err)
		}
	}

	s := MetaCacheStatsSnapshot()
	if s.Bypasses != 2 {
		t.Fatalf("disabled cache must bypass, got %+v", s)
	}
	if s.Fetches != 3 || s.Hits != 1 {
		t.Fatalf("bypass must not touch cache counters: %+v", s)
	}
	if s.HitRate() <= 0 {
		t.Fatalf("expected a positive hit rate, got %+v", s)
	}
}
