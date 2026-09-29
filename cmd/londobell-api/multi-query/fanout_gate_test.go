package multiquery

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// ---------------------------------------------------------------------------
// 可控的假分片：不连库，只记录「同时在飞的查询数 / 同时持有的缓冲区字节数」，
// 并支持注入延迟与错误。所有闸门测试都建立在它之上。
// ---------------------------------------------------------------------------

type fakeShard struct {
	mu            sync.Mutex
	inflight      int
	peak          int
	liveBytes     int64
	peakLiveBytes int64
	calls         []string
	results       map[string][]bson.M
	errs          map[string]error
	delay         func(desc string) time.Duration
	bufBytes      int
	entered       chan string // 可选：每次进入查询时发一个 desc（用于同步）
}

func (f *fakeShard) query(ctx context.Context, col *mongo.Collection, _ interface{}, _ bool) ([]bson.M, error) {
	desc := columnDesc(col)

	f.mu.Lock()
	f.inflight++
	if f.inflight > f.peak {
		f.peak = f.inflight
	}
	f.calls = append(f.calls, desc)
	if f.bufBytes > 0 {
		f.liveBytes += int64(f.bufBytes)
		if f.liveBytes > f.peakLiveBytes {
			f.peakLiveBytes = f.liveBytes
		}
	}
	d := time.Duration(0)
	if f.delay != nil {
		d = f.delay(desc)
	}
	err := f.errs[desc]
	res := f.results[desc]
	entered := f.entered
	f.mu.Unlock()

	if entered != nil {
		select {
		case entered <- desc:
		default:
		}
	}

	// 真实分配一块内存并在「慢分片」期间一直持有：模型与线上一致 ——
	// 内存峰值来自「同时在飞的 cur.All() 结果缓冲区」。
	var buf []byte
	if f.bufBytes > 0 {
		buf = make([]byte, f.bufBytes)
		buf[0] = 1
	}

	if d > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(d):
		}
	}
	runtime.KeepAlive(buf)

	f.mu.Lock()
	f.inflight--
	f.liveBytes -= int64(f.bufBytes)
	f.mu.Unlock()

	if err != nil {
		return nil, err
	}

	// 返回深拷贝：让「闸门开 / 闸门关」两次运行各自拿到独立副本，
	// 结果比较才有意义（否则两次都指向同一份底层数据）。
	return cloneShardResult(res), nil
}

func (f *fakeShard) stats() (peak int, calls int, peakLiveBytes int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peak, len(f.calls), f.peakLiveBytes
}

func cloneShardResult(in []bson.M) []bson.M {
	if in == nil {
		return nil
	}

	out := make([]bson.M, len(in))
	for i, m := range in {
		c := make(bson.M, len(m))
		for k, v := range m {
			c[k] = v
		}
		out[i] = c
	}

	return out
}

func installFakeShard(t *testing.T, f *fakeShard) {
	t.Helper()

	prev := shardQueryFunc
	shardQueryFunc = f.query
	t.Cleanup(func() { shardQueryFunc = prev })
}

func useGate(t *testing.T, limit int, wait time.Duration) {
	t.Helper()

	SetShardGateLimit(limit, wait)
	t.Cleanup(func() { SetShardGateLimit(DefaultShardQueryConcurrency, DefaultShardQueryWait) })
}

func coldLibs(t *testing.T, n int, colNames ...string) []CountUtil {
	t.Helper()

	if len(colNames) == 0 {
		colNames = []string{"ExecTrace"}
	}

	specs := make([]dbSpec, 0, n)
	for i := 0; i < n; i++ {
		specs = append(specs, dbSpec{name: fmt.Sprintf("cold_%02d", i), dtype: smodel.Cold, cols: colNames})
	}

	return buildCountLists(t, specs)
}

// 无 ctx 引用的合法管道源码（util.Parse 用 otto 求值）。
const testAggregator = `[{"$match": {"IsBlock": true}}]`

func newTestReq() model.CommonReq { return model.CommonReq{} }

// rangeLibs 给各库安排递减的高度区间，使 MultiRangeQuery 恰好为每个库生成一个
// aggList（顺序与 countLists 一致），便于断言结果顺序。
func rangeLibs(countLists []CountUtil) []CountUtil {
	out := make([]CountUtil, len(countLists))
	copy(out, countLists)
	for i := range out {
		out[i].Start = int64(200 - 100*i)
		out[i].End = out[i].Start + 100
	}

	return out
}

// tipsetLibs 给各库填上 TipSetStates 计数，使 MultiPagingQuery 为每个库生成一个 aggList。
func tipsetLibs(countLists []CountUtil) []CountUtil {
	out := make([]CountUtil, len(countLists))
	copy(out, countLists)
	for i := range out {
		out[i].TipSetStates = 100
	}

	return out
}

// ---------------------------------------------------------------------------
// ① 限流器把在飞查询数压在 N 以下，且一条查询都不许丢
// ---------------------------------------------------------------------------

func TestShardGateCapsInFlightShardQueries(t *testing.T) {
	const (
		limit    = 4
		libs     = 12
		requests = 6
	)

	fake := &fakeShard{delay: func(string) time.Duration { return 25 * time.Millisecond }}
	installFakeShard(t, fake)
	useGate(t, limit, 5*time.Second)

	countLists := coldLibs(t, libs)

	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := MultiTraversalQuery(context.Background(), []bson.M{}, countLists, "ExecTrace"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	peak, calls, _ := fake.stats()

	if peak > limit {
		t.Fatalf("peak in-flight shard queries = %d, want <= %d", peak, limit)
	}
	if peak == 0 {
		t.Fatalf("fake shard was never entered; test is not exercising the fan-out path")
	}
	if want := requests * libs; calls != want {
		t.Fatalf("shard queries executed = %d, want %d (no query may be dropped under normal load)", calls, want)
	}
}

// ---------------------------------------------------------------------------
// ② 慢分片场景：行为必须是「显式报错」，绝不是「静默截断 / 返回空」
// ---------------------------------------------------------------------------

func TestSlowShardFailsExplicitlyInsteadOfTruncating(t *testing.T) {
	fake := &fakeShard{delay: func(string) time.Duration { return 400 * time.Millisecond }}
	installFakeShard(t, fake)

	// 1 个令牌 + 40ms 有界等待 ⇒ 慢分片占住令牌时，后面的查询必然等不到。
	useGate(t, 1, 40*time.Millisecond)

	countLists := coldLibs(t, 3)

	res, err := MultiTraversalQuery(context.Background(), []bson.M{}, countLists, "ExecTrace")
	if err == nil {
		t.Fatalf("want an explicit error under shard slowdown, got nil (res=%v)", res)
	}
	if !IsFanoutSaturated(err) {
		t.Fatalf("want FanoutSaturatedError, got %T: %v", err, err)
	}
	if res != nil {
		t.Fatalf("must NOT return a (possibly partial) result on saturation, got %v", res)
	}
}

// 拿不到令牌时也绝不能把结果当成「这个库没有数据」：错误必须一路冒泡到调用方。
func TestShardGateSaturationPropagatesThroughEveryFanoutEntryPoint(t *testing.T) {
	fake := &fakeShard{delay: func(string) time.Duration { return 300 * time.Millisecond }}
	installFakeShard(t, fake)

	useGate(t, 1, 30*time.Millisecond)

	libs := coldLibs(t, 3)
	req := newTestReq()

	cases := []struct {
		name string
		run  func() ([]bson.M, error)
	}{
		{"MultiTraversalQuery", func() ([]bson.M, error) {
			return MultiTraversalQuery(context.Background(), []bson.M{}, libs, "ExecTrace")
		}},
		{"MultiUnionQuery", func() ([]bson.M, error) {
			return MultiUnionQuery(context.Background(), []bson.M{}, libs, "ExecTrace")
		}},
		{"MultiRangeQuery", func() ([]bson.M, error) {
			return MultiRangeQuery(context.Background(), 50, 300, rangeLibs(libs), []byte(testAggregator), req, "ExecTrace")
		}},
		{"MultiPagingQuery", func() ([]bson.M, error) {
			return MultiPagingQuery(context.Background(), 0, 0, TipSetStates, tipsetLibs(libs), []byte(testAggregator), req, "ExecTrace")
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// 每次先把闸门占满（一个 300ms 的慢查询），再发目标请求。
			blocker, err := shardGate.acquire(context.Background())
			if err != nil {
				t.Fatalf("acquire blocker slot: %v", err)
			}
			defer blocker()

			res, err := tc.run()
			if err == nil {
				t.Fatalf("want explicit error, got nil (res=%v)", res)
			}
			if !IsFanoutSaturated(err) {
				t.Fatalf("want FanoutSaturatedError, got %T: %v", err, err)
			}
			if res != nil {
				t.Fatalf("must not return partial result, got %v", res)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ③ 正常路径结果不变：闸门开 / 关两次运行逐元素相等（含顺序），
//    且与独立算出的期望顺序一致。
// ---------------------------------------------------------------------------

func TestFanoutResultsIdenticalWithGateOnAndOff(t *testing.T) {
	results := map[string][]bson.M{
		"cold_00.ExecTrace": {{"i": int32(1)}, {"i": int32(2)}},
		// 其余库返回空：MultiTraversalQuery 是「第一个非空结果胜出」，
		// 多于一个非空库时胜者随调度变化（既有语义，非本改动引入），
		// 因此这里只留一个非空库让结果可判定。
		"cold_01.ExecTrace": {},
		"cold_02.ExecTrace": {},
	}
	// MultiTraversalQuery 的语义是「取第一个非空结果」：只有 cold_00 非空，
	// 因此期望就是 cold_00 的那两行（顺序敏感）。
	wantTraversal := []bson.M{{"i": int32(1)}, {"i": int32(2)}}

	run := func(t *testing.T, gateLimit int) []bson.M {
		t.Helper()

		fake := &fakeShard{results: results}
		installFakeShard(t, fake)
		useGate(t, gateLimit, 5*time.Second)

		res, err := MultiTraversalQuery(context.Background(), []bson.M{}, coldLibs(t, 3), "ExecTrace")
		if err != nil {
			t.Fatalf("gateLimit=%d: unexpected error: %v", gateLimit, err)
		}
		return res
	}

	off := run(t, 0)     // 关闭闸门 = 改前行为（直通执行）
	on := run(t, 256)    // 开启闸门
	onSmall := run(t, 1) // 闸门很紧但负载很轻：仍必须完整、有序

	if !reflect.DeepEqual(off, wantTraversal) {
		t.Fatalf("gate off result = %v, want %v", off, wantTraversal)
	}
	if !reflect.DeepEqual(on, off) {
		t.Fatalf("gate on result = %v, want identical to gate off %v", on, off)
	}
	if !reflect.DeepEqual(onSmall, off) {
		t.Fatalf("tight gate result = %v, want identical to gate off %v", onSmall, off)
	}
}

func TestUnionRangeAndPagingResultsIdenticalWithGateOnAndOff(t *testing.T) {
	results := map[string][]bson.M{
		"cold_00.ExecTrace": {{"i": int32(1)}},
		"cold_01.ExecTrace": {{"i": int32(2)}, {"i": int32(3)}},
		"cold_02.ExecTrace": {{"i": int32(4)}},
	}
	wantConcat := []bson.M{{"i": int32(1)}, {"i": int32(2)}, {"i": int32(3)}, {"i": int32(4)}}

	type scenario struct {
		name string
		run  func(t *testing.T, countLists []CountUtil) []bson.M
	}

	scenarios := []scenario{
		{
			name: "MultiUnionQuery",
			run: func(t *testing.T, countLists []CountUtil) []bson.M {
				res, err := MultiUnionQuery(context.Background(), []bson.M{}, countLists, "ExecTrace")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return res
			},
		},
		{
			name: "MultiRangeQuery",
			run: func(t *testing.T, countLists []CountUtil) []bson.M {
				res, err := MultiRangeQuery(context.Background(), 50, 300, rangeLibs(countLists), []byte(testAggregator), newTestReq(), "ExecTrace")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return res
			},
		},
		{
			name: "MultiPagingQuery",
			run: func(t *testing.T, countLists []CountUtil) []bson.M {
				res, err := MultiPagingQuery(context.Background(), 0, 0, TipSetStates, tipsetLibs(countLists), []byte(testAggregator), newTestReq(), "ExecTrace")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return res
			},
		},
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			run := func(t *testing.T, gateLimit int) []bson.M {
				t.Helper()

				fake := &fakeShard{results: results}
				installFakeShard(t, fake)
				useGate(t, gateLimit, 5*time.Second)

				return sc.run(t, coldLibs(t, 3))
			}

			off := run(t, 0)
			on := run(t, 256)

			if !reflect.DeepEqual(off, wantConcat) {
				t.Fatalf("gate off result = %v, want %v", off, wantConcat)
			}
			if !reflect.DeepEqual(on, off) {
				t.Fatalf("gate on result = %v, want identical to gate off %v", on, off)
			}
		})
	}
}

// cid 两阶段路径的「完整管道」也必须走同一份全局预算。
func TestCidAggregatePathUsesGlobalGate(t *testing.T) {
	entered := make(chan string, 8)
	fake := &fakeShard{
		delay:   func(string) time.Duration { return 250 * time.Millisecond },
		entered: entered,
	}
	installFakeShard(t, fake)
	useGate(t, 1, 30*time.Millisecond)

	col := testClient(t).Database("cold_01").Collection("ExecTrace")

	first := make(chan error, 1)
	go func() {
		_, err := mongoCidAggregate(context.Background(), col, []bson.M{}, "ExecTrace")
		first <- err
	}()

	// 等第一个真的进入（占住唯一的令牌），再发第二个。
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first aggregate never entered the fake shard")
	}

	if _, err := mongoCidAggregate(context.Background(), col, []bson.M{}, "ExecTrace"); err == nil || !IsFanoutSaturated(err) {
		t.Fatalf("second cid aggregate must fail explicitly on a saturated gate, got %v", err)
	}

	if err := <-first; err != nil {
		t.Fatalf("first cid aggregate: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ④ 请求级闸（B 方案）：默认关闭 = 行为不变；开启后超限显式 503 语义
// ---------------------------------------------------------------------------

func TestFanoutRequestGateDisabledByDefault(t *testing.T) {
	SetFanoutRequestGateLimit(0)
	t.Cleanup(func() { SetFanoutRequestGateLimit(0) })

	for i := 0; i < 200; i++ {
		release, err := AcquireFanoutRequest()
		if err != nil {
			t.Fatalf("gate must be a no-op when disabled, got %v at iteration %d", err, i)
		}
		release()
	}

	if got := FanoutRequestGateSnapshot().Rejected; got != 0 {
		t.Fatalf("rejected = %d, want 0 when disabled", got)
	}
}

func TestFanoutRequestGateRejectsWhenSaturated(t *testing.T) {
	SetFanoutRequestGateLimit(2)
	t.Cleanup(func() { SetFanoutRequestGateLimit(0) })

	rel1, err := AcquireFanoutRequest()
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	rel2, err := AcquireFanoutRequest()
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}

	_, err = AcquireFanoutRequest()
	if err == nil {
		t.Fatal("third acquire must be rejected")
	}
	var rerr *RequestGateSaturatedError
	if !errors.As(err, &rerr) {
		t.Fatalf("want RequestGateSaturatedError, got %T: %v", err, err)
	}
	if rerr.Limit != 2 {
		t.Fatalf("reported limit = %d, want 2", rerr.Limit)
	}

	rel1()

	rel3, err := AcquireFanoutRequest()
	if err != nil {
		t.Fatalf("slot must be reusable after release, got %v", err)
	}
	if got := FanoutRequestGateSnapshot().InFlight; got != 2 {
		t.Fatalf("inflight = %d, want 2", got)
	}

	rel2()
	rel3()
	if got := FanoutRequestGateSnapshot().InFlight; got != 0 {
		t.Fatalf("inflight = %d, want 0 after releasing everything", got)
	}
}

// ---------------------------------------------------------------------------
// ⑤ 策略解析：环境变量 > 配置文件 > 内置默认
// ---------------------------------------------------------------------------

func TestResolveShardGatePolicy(t *testing.T) {
	t.Setenv(ShardQueryConcurrencyEnv, "")
	t.Setenv(ShardQueryWaitSecondsEnv, "")

	cases := []struct {
		name      string
		cfg       common.Config
		envLimit  string
		envWait   string
		wantLimit int
		wantWait  time.Duration
	}{
		{name: "zero_value_uses_defaults", cfg: common.Config{}, wantLimit: DefaultShardQueryConcurrency, wantWait: DefaultShardQueryWait},
		{name: "config_positive", cfg: common.Config{ShardQueryConcurrency: 64, ShardQueryWaitSeconds: 1}, wantLimit: 64, wantWait: time.Second},
		{name: "config_negative_disables", cfg: common.Config{ShardQueryConcurrency: -1, ShardQueryWaitSeconds: -1}, wantLimit: 0, wantWait: 0},
		{name: "env_overrides_config", cfg: common.Config{ShardQueryConcurrency: 64, ShardQueryWaitSeconds: 1}, envLimit: "128", envWait: "2", wantLimit: 128, wantWait: 2 * time.Second},
		{name: "env_zero_disables", cfg: common.Config{ShardQueryConcurrency: 64}, envLimit: "0", wantLimit: 0, wantWait: DefaultShardQueryWait},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(ShardQueryConcurrencyEnv, tc.envLimit)
			t.Setenv(ShardQueryWaitSecondsEnv, tc.envWait)

			gotLimit, gotWait := ResolveShardGatePolicy(tc.cfg)
			if gotLimit != tc.wantLimit {
				t.Fatalf("limit = %d, want %d", gotLimit, tc.wantLimit)
			}
			if gotWait != tc.wantWait {
				t.Fatalf("wait = %v, want %v", gotWait, tc.wantWait)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ⑥ 可复现的内存对比测量：闸门关 vs 开，同等并发下「同时在飞的结果缓冲区」峰值
//    （用真实分配 + 人为延迟模拟慢分片；不需要 mongo）
// ---------------------------------------------------------------------------

func TestFanoutGateBoundsPeakHeldBuffers(t *testing.T) {
	const (
		requests = 12
		libs     = 12
		bufBytes = 1 << 20 // 每次分片查询持有一个 1MiB 缓冲区
		gate     = 8
	)

	measure := func(t *testing.T, gateLimit int) (peakInflight int, peakHeldBytes int64, peakHeapInuse uint64) {
		t.Helper()

		fake := &fakeShard{
			bufBytes: bufBytes,
			delay:    func(string) time.Duration { return 25 * time.Millisecond },
		}
		installFakeShard(t, fake)
		useGate(t, gateLimit, 10*time.Second)

		countLists := coldLibs(t, libs)

		runtime.GC()
		stop := make(chan struct{})
		var (
			swg     sync.WaitGroup
			heapMax uint64
		)
		swg.Add(1)
		go func() {
			defer swg.Done()
			var ms runtime.MemStats
			for {
				select {
				case <-stop:
					return
				default:
				}

				runtime.ReadMemStats(&ms)
				if ms.HeapInuse > heapMax {
					heapMax = ms.HeapInuse
				}
				time.Sleep(2 * time.Millisecond)
			}
		}()

		var wg sync.WaitGroup
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = MultiTraversalQuery(context.Background(), []bson.M{}, countLists, "ExecTrace")
			}()
		}
		wg.Wait()

		close(stop)
		swg.Wait()

		peakInflight, _, peakHeldBytes = fake.stats()
		return peakInflight, peakHeldBytes, heapMax
	}

	// 先测「闸门开」（避免前一轮遗留堆把这一轮抬起来），再测「闸门关」。
	gPeak, gHeld, gHeap := measure(t, gate)
	uPeak, uHeld, uHeap := measure(t, 0)

	t.Logf("gate=off: peak_inflight=%d peak_held_buffers=%.1fMiB heap_inuse_peak=%.1fMiB", uPeak, float64(uHeld)/(1<<20), float64(uHeap)/(1<<20))
	t.Logf("gate=%d  : peak_inflight=%d peak_held_buffers=%.1fMiB heap_inuse_peak=%.1fMiB", gate, gPeak, float64(gHeld)/(1<<20), float64(gHeap)/(1<<20))

	if gPeak > gate {
		t.Fatalf("gated peak in-flight = %d, want <= %d", gPeak, gate)
	}
	if gHeld > int64(gate*bufBytes) {
		t.Fatalf("gated peak held buffers = %d bytes, want <= %d", gHeld, gate*bufBytes)
	}
	if uPeak <= gate {
		t.Fatalf("ungated peak in-flight = %d, want > %d (the test must actually reproduce the unbounded case)", uPeak, gate)
	}
	if gHeld >= uHeld {
		t.Fatalf("gate did not reduce held buffers: gated=%d ungated=%d", gHeld, uHeld)
	}
}
