package multiquery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// 真实链路（一次性 scratch mongod）：
//
//	MONGO_TEST_URI=mongodb://127.0.0.1:27099 go test ./cmd/londobell-api/multi-query/ \
//	    -run 'TestBounded|TestResultSize|TestRefreshStateIsGated|TestFanoutResultsByteIdentical' -v -timeout 15m
//
// 没设置 MONGO_TEST_URI 时整体跳过。绝不要指向生产/冷存储节点。

func mongoTestURI(t *testing.T) string {
	t.Helper()

	uri := strings.TrimSpace(os.Getenv("MONGO_TEST_URI"))
	if uri == "" {
		t.Skip("set MONGO_TEST_URI (scratch mongod, never production/cold storage) to run")
	}
	return uri
}

func mongoTestClient(t *testing.T, uri string) *mongo.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetMaxPoolSize(64))
	if err != nil {
		t.Fatalf("connect %v: %v", uri, err)
	}
	if err := cli.Ping(ctx, nil); err != nil {
		t.Fatalf("ping %v: %v", uri, err)
	}

	t.Cleanup(func() { _ = cli.Disconnect(context.Background()) })
	return cli
}

// seedDocs 往 db.col 灌 n 条 ~payloadBytes 的文档（带递增 Seq，便于断言顺序）。
func seedDocs(t *testing.T, cli *mongo.Client, db, col string, n, payloadBytes int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c := cli.Database(db).Collection(col)
	_ = cli.Database(db).Drop(ctx)

	payload := strings.Repeat("x", payloadBytes)
	batch := make([]interface{}, 0, 1000)
	for i := 0; i < n; i++ {
		batch = append(batch, bson.M{"Seq": int64(i), "IsBlock": true, "Epoch": i, "Blob": payload})
		if len(batch) == 1000 {
			if _, err := c.InsertMany(ctx, batch); err != nil {
				t.Fatalf("insert into %v.%v: %v", db, col, err)
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if _, err := c.InsertMany(ctx, batch); err != nil {
			t.Fatalf("insert into %v.%v: %v", db, col, err)
		}
	}

	t.Cleanup(func() { _ = cli.Database(db).Drop(context.Background()) })
}

func withResultSizeLimit(t *testing.T, n int64) {
	t.Helper()

	prev := common.MaxResultBytes()
	common.SetMaxResultBytes(n)
	t.Cleanup(func() { common.SetMaxResultBytes(prev) })
}

func jsonBytes(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// ---------------------------------------------------------------------------
// ① 正常负载：BoundedAll 与 cur.All 返回值逐字节一致（含顺序）
// ---------------------------------------------------------------------------

func TestBoundedAllMatchesCursorAllByteForByte(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()
	db, col := "rs_identity", "ExecTrace"
	seedDocs(t, cli, db, col, 500, 64)
	c := cli.Database(db).Collection(col)

	// bson.M（[]bson.M，fanout 路径的实际结果类型）
	cur1, err := c.Find(ctx, bson.D{})
	if err != nil {
		t.Fatalf("find #1: %v", err)
	}
	var want []bson.M
	if err := cur1.All(ctx, &want); err != nil {
		t.Fatalf("cur.All: %v", err)
	}

	withResultSizeLimit(t, common.DefaultMaxResultBytes)
	cur2, err := c.Find(ctx, bson.D{})
	if err != nil {
		t.Fatalf("find #2: %v", err)
	}
	var got []bson.M
	if err := common.BoundedAll(ctx, cur2, &got, "test"); err != nil {
		t.Fatalf("BoundedAll: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("DeepEqual mismatch: want %d docs, got %d docs", len(want), len(got))
	}
	if !bytes.Equal(jsonBytes(t, want), jsonBytes(t, got)) {
		t.Fatalf("JSON bytes differ (order or values changed)")
	}
	if len(got) != 500 {
		t.Fatalf("docs = %d, want 500", len(got))
	}
	// 顺序：Seq 必须严格递增。
	for i, d := range got {
		if seq, _ := d["Seq"].(int64); seq != int64(i) {
			t.Fatalf("order changed at %d: Seq=%v", i, d["Seq"])
		}
	}

	// 指针元素（[]*T，segment.GetDBState 的实际类型）也要一致。
	type doc struct {
		Seq     int64  `bson:"Seq"`
		IsBlock bool   `bson:"IsBlock"`
		Blob    string `bson:"Blob"`
	}

	cur3, _ := c.Find(ctx, bson.D{})
	var wantPtr []*doc
	if err := cur3.All(ctx, &wantPtr); err != nil {
		t.Fatalf("cur.All ptr: %v", err)
	}

	cur4, _ := c.Find(ctx, bson.D{})
	var gotPtr []*doc
	if err := common.BoundedAll(ctx, cur4, &gotPtr, "test"); err != nil {
		t.Fatalf("BoundedAll ptr: %v", err)
	}
	if !reflect.DeepEqual(wantPtr, gotPtr) {
		t.Fatalf("pointer-element mismatch")
	}
	for _, p := range gotPtr {
		if p == nil {
			t.Fatalf("nil pointer element: decoder did not allocate")
		}
	}
}

// 上限关闭（<=0）= 回退旧行为：与 cur.All 完全一致。
func TestBoundedAllDisabledFallsBackToCursorAll(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()
	db, col := "rs_disabled", "ExecTrace"
	seedDocs(t, cli, db, col, 300, 64)
	c := cli.Database(db).Collection(col)

	cur1, _ := c.Find(ctx, bson.D{})
	var want []bson.M
	if err := cur1.All(ctx, &want); err != nil {
		t.Fatalf("cur.All: %v", err)
	}

	for _, limit := range []int64{0, -1} {
		withResultSizeLimit(t, limit)

		cur2, _ := c.Find(ctx, bson.D{})
		var got []bson.M
		if err := common.BoundedAll(ctx, cur2, &got, "test"); err != nil {
			t.Fatalf("limit=%d: unexpected error: %v", limit, err)
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("limit=%d: result differs from cur.All", limit)
		}
	}
}

// ---------------------------------------------------------------------------
// ② 超限：显式错误，绝不是截断 / 部分结果
// ---------------------------------------------------------------------------

func TestBoundedAllOverLimitIsExplicitErrorNotTruncation(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()
	db, col := "rs_overlimit", "ExecTrace"
	seedDocs(t, cli, db, col, 500, 64)
	c := cli.Database(db).Collection(col)

	// 上限压到 1 字节：第一条文档就会超。
	withResultSizeLimit(t, 1)

	cur, err := c.Find(ctx, bson.D{})
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	var res []bson.M
	err = common.BoundedAll(ctx, cur, &res, "shard_query")
	if err == nil {
		t.Fatalf("want explicit error over limit, got %d docs", len(res))
	}
	if !common.IsResultTooLarge(err) {
		t.Fatalf("want ResultTooLargeError, got %T: %v", err, err)
	}
	if res != nil {
		t.Fatalf("must NOT return a (partial) result, got %d docs", len(res))
	}

	var rte *common.ResultTooLargeError
	if !errors.As(err, &rte) {
		t.Fatalf("errors.As failed for %T", err)
	}
	if rte.Op != "shard_query" || rte.Limit != 1 || rte.Actual <= 1 {
		t.Fatalf("unexpected error fields: %+v", *rte)
	}

	// 上限恰好卡在「部分文档之后」时也必须整体失败（不是截断到前 N 条）。
	// 先量出单条文档的 BSON 字节数，再把上限设成 ~10 条的字节数。
	withResultSizeLimit(t, 0)
	probe, err := c.Find(ctx, bson.D{})
	if err != nil {
		t.Fatalf("probe find: %v", err)
	}
	if !probe.Next(ctx) {
		t.Fatalf("probe: no docs (err=%v)", probe.Err())
	}
	docBytes := int64(len(probe.Current))
	_ = probe.Close(ctx)

	withResultSizeLimit(t, docBytes*10)
	cur4, _ := c.Find(ctx, bson.D{})
	var res4 []bson.M
	err = common.BoundedAll(ctx, cur4, &res4, "shard_query")
	if err == nil {
		t.Fatalf("want error at ~10 docs, got %d docs", len(res4))
	}
	if !common.IsResultTooLarge(err) {
		t.Fatalf("want ResultTooLargeError, got %T: %v", err, err)
	}
	if res4 != nil {
		t.Fatalf("must not return truncated prefix, got %d docs", len(res4))
	}
}

// ---------------------------------------------------------------------------
// ③ 端到端：cap 开 / 关，fanout 入口返回值逐字节一致
// ---------------------------------------------------------------------------

func TestFanoutResultsByteIdenticalWithCapOnAndOff(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()
	db, col := "rs_fanout_identity", "ExecTrace"
	seedDocs(t, cli, db, col, 400, 64)

	countLists := []CountUtil{{
		Cols:  common.Collections{DB: cli.Database(db), Cols: []*mongo.Collection{cli.Database(db).Collection(col)}},
		DType: smodel.Cold,
	}}

	useGate(t, DefaultShardQueryConcurrency, DefaultShardQueryWait)

	// 空管道 = 返回集合全部文档；必须是切片（driver 只接受 slice/array 当管道，
	// 与生产一致：pipe 来自 util.Parse 的 value2agg，返回的就是切片）。
	pipe := mongo.Pipeline{}

	run := func(t *testing.T, limit int64) []bson.M {
		t.Helper()
		withResultSizeLimit(t, limit)

		res, err := MultiTraversalQuery(ctx, pipe, countLists, "ExecTrace")
		if err != nil {
			t.Fatalf("limit=%d: unexpected error: %v", limit, err)
		}
		return res
	}

	off := run(t, 0)                           // 关闭上限 = 改前行为
	on := run(t, common.DefaultMaxResultBytes) // 默认上限（远大于本结果）
	onTight := run(t, 1<<30)                   // 1GiB：仍然宽松

	if !bytes.Equal(jsonBytes(t, off), jsonBytes(t, on)) {
		t.Fatalf("cap on vs off: JSON bytes differ")
	}
	if !bytes.Equal(jsonBytes(t, off), jsonBytes(t, onTight)) {
		t.Fatalf("tight cap vs off: JSON bytes differ")
	}
	if len(off) != 400 {
		t.Fatalf("docs = %d, want 400", len(off))
	}
}

// ---------------------------------------------------------------------------
// ④ mongoShardQuery 是 fanout 的唯一物化点：cap 通过闸门 + 游标生效
// ---------------------------------------------------------------------------

func TestMongoShardQueryEnforcesByteCap(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()
	db, col := "rs_shardquery_cap", "ExecTrace"
	seedDocs(t, cli, db, col, 500, 256)

	c := cli.Database(db).Collection(col)
	pipe := mongo.Pipeline{bson.D{{Key: "$match", Value: bson.D{{Key: "IsBlock", Value: true}}}}}

	useGate(t, DefaultShardQueryConcurrency, DefaultShardQueryWait)

	// 关闭上限：全量返回。
	withResultSizeLimit(t, 0)
	res, err := runShardQuery(ctx, c, pipe, false)
	if err != nil {
		t.Fatalf("cap off: %v", err)
	}
	if len(res) != 500 {
		t.Fatalf("cap off: docs = %d, want 500", len(res))
	}

	// 上限很小：显式报错，且不返回部分结果。
	withResultSizeLimit(t, 4096)
	res, err = runShardQuery(ctx, c, pipe, false)
	if err == nil {
		t.Fatalf("cap on: want error, got %d docs", len(res))
	}
	if !common.IsResultTooLarge(err) {
		t.Fatalf("cap on: want ResultTooLargeError, got %T: %v", err, err)
	}
	if res != nil {
		t.Fatalf("cap on: must not return partial result, got %d docs", len(res))
	}
}

// ---------------------------------------------------------------------------
// ⑤ 元数据刷新（RefreshState 路径）被纳入同一个全局闸
// ---------------------------------------------------------------------------

func TestRefreshStateIsGatedByShardGate(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()

	// --- 负向：闸门被占满 ⇒ RefreshState 显式报 FanoutSaturatedError，
	// 且在拿到令牌之前**不会**碰 mongo（所以 Segment 可以是零值）。
	useGate(t, 1, 30*time.Millisecond)
	blocker, err := shardGate.acquire(ctx)
	if err != nil {
		t.Fatalf("acquire blocker: %v", err)
	}
	defer blocker()

	dbsmEmpty := &DataBaseStateManager{
		Segment:      &segment.Segment{},
		DBStateCache: NewDataBaseStateCache(),
	}
	_, _, err = dbsmEmpty.RefreshState(ctx, "mongodb://127.0.0.1:1/none")
	if err == nil {
		t.Fatalf("want FanoutSaturatedError while gate is saturated, got nil")
	}
	if !IsFanoutSaturated(err) {
		t.Fatalf("want FanoutSaturatedError, got %T: %v", err, err)
	}

	// --- 正向：闸门打开 ⇒ 真实走完 Segment.GetState（7 次元数据 find）。
	useGate(t, DefaultShardQueryConcurrency, DefaultShardQueryWait)

	const segName = "rs_segstate"
	mds := datastore.NewMapDatastore()
	segMgr, err := segment.NewStateManager(mds)
	if err != nil {
		t.Fatalf("new state manager: %v", err)
	}
	if err := segMgr.SetActive(segName); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if err := segMgr.SetInfo(segName, segment.Info{Read: uri, Write: uri}); err != nil {
		t.Fatalf("set info: %v", err)
	}

	seg, err := segment.New(ctx, segMgr, common.Config{})
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}

	dsn := "scratch://rs-segstate"
	t.Cleanup(func() { _ = cli.Database(segName).Drop(context.Background()) })
	if _, err := cli.Database(segName).Collection("DBState").InsertOne(ctx, bson.M{
		"Dsn":        dsn,
		"StartEpoch": int64(0),
		"EndEpoch":   int64(100),
		"DType":      int(smodel.Cold),
		"Interval":   int64(100),
	}); err != nil {
		t.Fatalf("insert DBState: %v", err)
	}

	dbsm := &DataBaseStateManager{Segment: seg, DBStateCache: NewDataBaseStateCache()}

	before := ShardGateSnapshot().Acquired
	state, found, err := dbsm.RefreshState(ctx, dsn)
	if err != nil {
		t.Fatalf("RefreshState (gate open): %v", err)
	}
	if !found || state == nil {
		t.Fatalf("RefreshState: found=%v state=%v", found, state)
	}
	if got := ShardGateSnapshot().Acquired; got <= before {
		t.Fatalf("RefreshState did not acquire a shard gate slot (acquired %d -> %d)", before, got)
	}
}

// ---------------------------------------------------------------------------
// ⑥ 内存实测：同样的负载，cap 关 / 开 的峰值 HeapInuse 对比
// ---------------------------------------------------------------------------

func TestResultSizeCapBoundsPeakMemoryAgainstMongo(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	const (
		requests = 4
		docs     = 3000
		docBytes = 16384
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, col := "rs_mem", "ExecTrace"
	seedDocs(t, cli, db, col, docs, docBytes)

	countLists := []CountUtil{{
		Cols:  common.Collections{DB: cli.Database(db), Cols: []*mongo.Collection{cli.Database(db).Collection(col)}},
		DType: smodel.Cold,
	}}

	useGate(t, DefaultShardQueryConcurrency, DefaultShardQueryWait)

	pipe := mongo.Pipeline{}

	measure := func(t *testing.T, limit int64) (peakHeap uint64, explicitErrs int, okDocs int) {
		t.Helper()

		withResultSizeLimit(t, limit)
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
				time.Sleep(3 * time.Millisecond)
			}
		}()

		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			bad  int
			good int
		)
		start := make(chan struct{})
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				res, err := MultiTraversalQuery(ctx, pipe, countLists, "ExecTrace")
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					if common.IsResultTooLarge(err) {
						bad++
					} else {
						t.Errorf("unexpected error: %v", err)
					}
					return
				}
				good += len(res)
			}()
		}
		close(start)
		wg.Wait()

		close(stop)
		swg.Wait()

		return heapMax, bad, good
	}

	// cap 关（= 改前）：每个在飞查询物化 ~6MB，8 个并发叠加。
	offHeap, offErrs, offDocs := measure(t, 0)
	// cap 开（1MiB）：每个查询在 ~1MiB 处显式失败并释放游标。
	onHeap, onErrs, onDocs := measure(t, 1<<20)

	t.Logf("cap off: peak_heap_inuse=%.2f MiB, explicit_errors=%d, docs=%d",
		float64(offHeap)/(1<<20), offErrs, offDocs)
	t.Logf("cap on : peak_heap_inuse=%.2f MiB, explicit_errors=%d, docs=%d",
		float64(onHeap)/(1<<20), onErrs, onDocs)

	if offErrs != 0 {
		t.Fatalf("cap off: unexpected explicit errors = %d", offErrs)
	}
	if offDocs == 0 {
		t.Fatalf("cap off: no docs materialized; measurement meaningless")
	}
	if onErrs != requests {
		t.Fatalf("cap on: explicit errors = %d, want %d (every query must fail explicitly)", onErrs, requests)
	}
	if onDocs != 0 {
		t.Fatalf("cap on: docs = %d, want 0 (no partial results)", onDocs)
	}
	if onHeap >= offHeap {
		t.Fatalf("cap did not reduce peak HeapInuse: cap_on=%.2fMiB cap_off=%.2fMiB",
			float64(onHeap)/(1<<20), float64(offHeap)/(1<<20))
	}
}
