package multiquery

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// ---------------------------------------------------------------------------
// 「启动形状」回归：聚合器启动时必须加载的元数据（segment_state）**不能**被
// 查询档的 16 MiB 上限拒掉。
//
// 上一版（单档上限）的测试全绿但漏了这一类：所有测试用的都是小结果集，
// 而生产故障恰恰是**启动路径**上的元数据只超了 138 字节：
//
//	ERROR result set too large: op=segment_state, limit=16777216 bytes,
//	      exceeded at 16777354 bytes; refusing to materialize a partial result
//	ERROR cli error: result set too large: op=segment_state, ...
//	⇒ 聚合器启动 exit 1、崩溃循环。
//
// 这个文件构造「> 16 MiB 的元数据 + 真实 GetState/RefreshState 链路」，
// 断言：
//	① 用查询档上限卡元数据 ⇒ 复现生产故障（显式 ResultTooLargeError）；
//	② 用启动时真正生效的策略（ApplyResultSizePolicy(默认配置) = FirstLoad 第一步）
//	   ⇒ 加载成功（修复点）；
//	③ 同一批数据走**扇出/查询档**仍然被 16 MiB 卡住（封顶意义没有被削弱）；
//	④ 字节记账与实测一致（limit == 实测字节 ⇒ 通过；limit == 实测-1 ⇒ 恰好拒绝）。
//
// 需要 scratch mongod：MONGO_TEST_URI=mongodb://127.0.0.1:27099
// 绝不要指向生产/冷存储节点。
// ---------------------------------------------------------------------------

// startupMetadataDoc 构造一条与生产同形状的 segment 元数据文档
// （model.SegmentState 的实际字段）。
func startupMetadataDoc(dsn string, i int) bson.M {
	return bson.M{
		"_id":        fmt.Sprintf("%s-actor-%d", dsn, i),
		"Dsn":        dsn,
		"StartEpoch": int64(i * 100),
		"EndEpoch":   int64((i + 1) * 100),
		"Count":      int64(1000 + i),
		"ActorID":    fmt.Sprintf("f0%04d", i),
		"MethodName": "Send",
	}
}

// seedStartupMetadata 灌入「启动形状」的元数据：DBState 一条 + ActorState 大量
// （>16 MiB，即生产故障量级）+ 其余元数据集合少量，覆盖 GetState 的全部 7 次 find。
// 返回 ActorState 的计划条数与单条字节数。
func seedStartupMetadata(t *testing.T, ctx context.Context, cli *mongo.Client, db, dsn string) (int, int64) {
	t.Helper()

	raw, err := bson.Marshal(startupMetadataDoc(dsn, 0))
	if err != nil {
		t.Fatalf("marshal sample metadata doc: %v", err)
	}
	perDoc := int64(len(raw))

	// 目标：比查询档默认上限（16 MiB）多出 ~20% 余量 —— 生产故障只多 138 字节，
	// 这里故意给足余量，保证「> 16 MiB」在任何 mongod/BSON 版本下都成立。
	target := common.DefaultMaxResultBytes * 6 / 5
	n := int(target/perDoc) + 1

	t.Logf("startup-shape metadata: per_doc=%d bytes, docs=%d, planned_actor_state_bytes=%d (%.2f MiB), query_default_limit=%d",
		perDoc, n, perDoc*int64(n), float64(perDoc*int64(n))/(1<<20), common.DefaultMaxResultBytes)

	database := cli.Database(db)
	_ = database.Drop(ctx)
	t.Cleanup(func() { _ = cli.Database(db).Drop(context.Background()) })

	// DBState：GetState 的第一步，必须存在（否则 found=false 直接返回）。
	if _, err := database.Collection("DBState").InsertOne(ctx, bson.M{
		"Dsn":        dsn,
		"StartEpoch": int64(0),
		"EndEpoch":   int64(100000000),
		"DType":      int(smodel.Cold),
		"Interval":   int64(100),
	}); err != nil {
		t.Fatalf("insert DBState: %v", err)
	}

	// ActorState：>16 MiB 的那一份（生产里就是它撑到 16,777,354 字节）。
	const batchSize = 20000
	batch := make([]interface{}, 0, batchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if _, err := database.Collection("ActorState").InsertMany(ctx, batch); err != nil {
			t.Fatalf("insert ActorState: %v", err)
		}
		batch = batch[:0]
	}
	for i := 0; i < n; i++ {
		batch = append(batch, startupMetadataDoc(dsn, i))
		if len(batch) == batchSize {
			flush()
		}
	}
	flush()

	// 其余元数据集合：小量，但让「7 次 find」的启动形状完整。
	for _, col := range []string{"BlockState", "BlockMethodState", "ActorMethodState", "ActorTransferState", "MinedState", "LargeAmountTransferState"} {
		small := make([]interface{}, 0, 200)
		for i := 0; i < 200; i++ {
			doc := startupMetadataDoc(dsn, i)
			doc["_id"] = fmt.Sprintf("%s-%s-%d", dsn, col, i)
			small = append(small, doc)
		}
		if _, err := database.Collection(col).InsertMany(ctx, small); err != nil {
			t.Fatalf("insert %v: %v", col, err)
		}
	}

	return n, perDoc
}

// measureMetadataBytes 用与生产**完全同口径**（累计 len(cur.Current)，mongo 返回的
// 原始 BSON 字节）量出该 dsn 的 ActorState 元数据字节数 —— 也就是
// op=segment_state 时 GetState 会累计的字节数。
func measureMetadataBytes(t *testing.T, ctx context.Context, cli *mongo.Client, db, dsn string) int64 {
	t.Helper()

	cur, err := cli.Database(db).Collection("ActorState").Find(ctx, bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		t.Fatalf("find ActorState: %v", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var total, docs int64
	for cur.Next(ctx) {
		total += int64(len(cur.Current))
		docs++
	}
	if err := cur.Err(); err != nil {
		t.Fatalf("cursor err: %v", err)
	}
	t.Logf("measured metadata bytes (op=segment_state accounting): %d bytes (%.3f MiB) over %d docs",
		total, float64(total)/(1<<20), docs)

	return total
}

func newSegForStartupTest(t *testing.T, ctx context.Context, uri, segName string) (*segment.Segment, *DataBaseStateManager) {
	t.Helper()

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

	return seg, &DataBaseStateManager{Segment: seg, DBStateCache: NewDataBaseStateCache()}
}

func TestStartupShapeMetadataLoadNotRejectedByQueryLimit(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	const segName = "rs_startup_metadata"
	dsn := "scratch://rs-startup-metadata"

	// 还原全局策略（本测试会改两档上限）。
	prevQ, prevM := common.MaxResultBytes(), common.MetadataMaxResultBytes()
	t.Cleanup(func() {
		common.SetMaxResultBytes(prevQ)
		common.SetMetadataMaxResultBytes(prevM)
	})

	seedStartupMetadata(t, ctx, cli, segName, dsn)
	_, dbsm := newSegForStartupTest(t, ctx, uri, segName)

	useGate(t, DefaultShardQueryConcurrency, DefaultShardQueryWait)

	measured := measureMetadataBytes(t, ctx, cli, segName, dsn)
	if measured <= common.DefaultMaxResultBytes {
		t.Fatalf("test setup broken: measured metadata %d must exceed the query default limit %d",
			measured, common.DefaultMaxResultBytes)
	}

	// ---- ① 复现生产故障：元数据被**查询档**上限（16 MiB）卡住。 ----
	// 这就是上一版的语义（单档），也正是生产崩溃循环的原因。
	common.SetMetadataMaxResultBytes(common.DefaultMaxResultBytes)
	_, _, err := dbsm.RefreshState(ctx, dsn)
	if err == nil {
		t.Fatalf("① metadata capped by the query-tier limit: want explicit ResultTooLargeError, got nil "+
			"(measured metadata %d bytes > limit %d)", measured, common.DefaultMaxResultBytes)
	}
	if !common.IsResultTooLarge(err) {
		t.Fatalf("① want ResultTooLargeError, got %T: %v", err, err)
	}
	var rte *common.ResultTooLargeError
	if ok := asResultTooLarge(err, &rte); !ok {
		t.Fatalf("① errors.As failed for %T", err)
	}
	if rte.Op != common.OpSegmentState {
		t.Fatalf("① op = %q, want %q (the startup metadata op)", rte.Op, common.OpSegmentState)
	}
	if rte.Limit != common.DefaultMaxResultBytes {
		t.Fatalf("① limit = %d, want %d", rte.Limit, common.DefaultMaxResultBytes)
	}
	if rte.Actual <= common.DefaultMaxResultBytes {
		t.Fatalf("① actual = %d, want > %d", rte.Actual, common.DefaultMaxResultBytes)
	}
	t.Logf("① reproduced production failure shape: %v", err)

	// ---- ④ 字节记账一致性：limit == 实测 ⇒ 通过；limit == 实测-1 ⇒ 恰好拒绝。 ----
	common.SetMetadataMaxResultBytes(measured)
	if _, _, err := dbsm.RefreshState(ctx, dsn); err != nil {
		t.Fatalf("④ metadata limit == measured (%d): want success, got %v", measured, err)
	}
	common.SetMetadataMaxResultBytes(measured - 1)
	_, _, err = dbsm.RefreshState(ctx, dsn)
	if err == nil || !common.IsResultTooLarge(err) {
		t.Fatalf("④ metadata limit == measured-1 (%d): want ResultTooLargeError, got %T: %v", measured-1, err, err)
	}
	if ok := asResultTooLarge(err, &rte); !ok || rte.Actual != measured {
		t.Fatalf("④ reported actual = %d, want exactly the measured %d (accounting must match)", rte.Actual, measured)
	}
	t.Logf("④ byte accounting matches measurement exactly: %d bytes", measured)

	// ---- ② 启动真正生效的策略（FirstLoad 的第一步）⇒ 元数据加载成功。 ----
	common.ApplyResultSizePolicy(common.Config{})
	if common.MetadataMaxResultBytes() != common.DefaultMetadataMaxResultBytes {
		t.Fatalf("② default policy: metadata limit = %d, want %d",
			common.MetadataMaxResultBytes(), common.DefaultMetadataMaxResultBytes)
	}
	if common.MaxResultBytes() != common.DefaultMaxResultBytes {
		t.Fatalf("② default policy: query limit = %d, want %d",
			common.MaxResultBytes(), common.DefaultMaxResultBytes)
	}

	state, found, err := dbsm.RefreshState(ctx, dsn)
	if err != nil {
		t.Fatalf("② startup-shape metadata load (%d bytes, limit %d) must succeed, got %v",
			measured, common.MetadataMaxResultBytes(), err)
	}
	if !found || state == nil {
		t.Fatalf("② startup-shape metadata load: found=%v state=%v", found, state)
	}
	if got := len(state.GetActorStates("f00001")); got == 0 {
		t.Fatalf("② startup-shape metadata load returned empty actor states")
	}
	t.Logf("② startup-shape metadata load OK: %d bytes loaded, metadata_limit=%d, query_limit=%d",
		measured, common.MetadataMaxResultBytes(), common.MaxResultBytes())

	// ---- ③ 扇出/查询档**仍然**被 16 MiB 卡住：封顶的意义没有被削弱。 ----
	// 同一批数据（ActorState，>16 MiB）走 shard_query op ⇒ 必须显式报错。
	res, err := runShardQuery(ctx, cli.Database(segName).Collection("ActorState"), mongo.Pipeline{}, false)
	if err == nil {
		t.Fatalf("③ shard_query over %d bytes must still be rejected by the query-tier limit, got %d docs",
			measured, len(res))
	}
	if !common.IsResultTooLarge(err) {
		t.Fatalf("③ want ResultTooLargeError, got %T: %v", err, err)
	}
	if ok := asResultTooLarge(err, &rte); !ok || rte.Op != "shard_query" || rte.Limit != common.DefaultMaxResultBytes {
		t.Fatalf("③ unexpected error fields: %+v (want op=shard_query limit=%d)", rte, common.DefaultMaxResultBytes)
	}
	if res != nil {
		t.Fatalf("③ must not return a partial result, got %d docs", len(res))
	}
	t.Logf("③ query-tier cap still enforced on the same data: %v", err)
}

// 「接近上限」告警必须真的发出来（元数据随链增长时的可观测信号）。
// 用真实 mongo 游标驱动 BoundedAll，把阈值卡在中间，断言 hook 被调用一次。
func TestBoundedAllWarnsWhenApproachingLimit(t *testing.T) {
	uri := mongoTestURI(t)
	cli := mongoTestClient(t, uri)

	ctx := context.Background()
	db, col := "rs_warn_threshold", "ActorState"
	seedDocs(t, cli, db, col, 400, 64)

	c := cli.Database(db).Collection(col)

	cur, err := c.Find(ctx, bson.D{})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !cur.Next(ctx) {
		t.Fatalf("probe: no docs")
	}
	docBytes := int64(len(cur.Current))
	_ = cur.Close(ctx)

	// 阈值设在「略高于整批结果」处：既会触发告警（>=80%），又不会超限。
	totalApprox := docBytes * 400
	limit := totalApprox + docBytes
	if limit <= 0 {
		t.Fatalf("bad probe: docBytes=%d", docBytes)
	}

	prevQ := common.MaxResultBytes()
	prevHook := common.ResultSizeWarnHook
	t.Cleanup(func() {
		common.SetMaxResultBytes(prevQ)
		common.ResultSizeWarnHook = prevHook
	})

	common.SetMaxResultBytes(limit)

	var (
		calls   int
		lastOp  string
		lastUse int64
		lastLim int64
	)
	common.ResultSizeWarnHook = func(op string, used, l int64) {
		calls++
		lastOp, lastUse, lastLim = op, used, l
	}

	cur2, _ := c.Find(ctx, bson.D{})
	var res []bson.M
	if err := common.BoundedAll(ctx, cur2, &res, "shard_query"); err != nil {
		t.Fatalf("BoundedAll: %v", err)
	}
	if len(res) != 400 {
		t.Fatalf("docs = %d, want 400", len(res))
	}
	if calls != 1 {
		t.Fatalf("warn hook calls = %d, want exactly 1 (fire once per materialization)", calls)
	}
	if lastOp != "shard_query" || lastLim != limit {
		t.Fatalf("warn hook args: op=%q limit=%d, want shard_query/%d", lastOp, lastLim, limit)
	}
	if !common.NearLimit(lastUse, lastLim) {
		t.Fatalf("warn fired at used=%d limit=%d which is not near limit", lastUse, lastLim)
	}
	t.Logf("warn hook fired at used=%d / limit=%d (%.1f%%)", lastUse, lastLim, 100*float64(lastUse)/float64(lastLim))

	// 关闭上限 ⇒ 不告警。
	common.SetMaxResultBytes(0)
	cur3, _ := c.Find(ctx, bson.D{})
	var res3 []bson.M
	if err := common.BoundedAll(ctx, cur3, &res3, "shard_query"); err != nil {
		t.Fatalf("BoundedAll (limit off): %v", err)
	}
	if calls != 1 {
		t.Fatalf("limit off must not warn: calls = %d, want 1", calls)
	}
}

// asResultTooLarge 是 errors.As 的小包装（保持调用点简洁）。
func asResultTooLarge(err error, target **common.ResultTooLargeError) bool {
	return errors.As(err, target)
}
