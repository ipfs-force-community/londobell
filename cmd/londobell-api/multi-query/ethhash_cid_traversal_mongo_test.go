package multiquery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// /aggregators/hash_by_messagecid（EthHash）两阶段改造的真库验证。
//
// 需要一个「一次性」的 mongod（本地 scratch 实例即可，绝不要指向生产/冷库）：
//
//	MONGO_TEST_URI=mongodb://127.0.0.1:27099 go test ./cmd/londobell-api/multi-query/ \
//	    -run TestMultiTraversalQueryByCidOnEthHashAgainstMongo -v
//
// 没设置 MONGO_TEST_URI 时整体跳过，保证 `go test ./...` 在没有 mongo 的环境上依然全绿。
//
// 它验证三件假实现里验不到的事：
//  1. EthHash 的探测在「生产索引形态」Cid_1_Epoch_1（sparse 复合索引）下依然是
//     PROJECTION_COVERED：keysExamined=1、docsExamined=0、计划里没有 FETCH
//     （这是整个改造省掉 11 次文档 FETCH 的前提，必须用真实计划复核）；
//  2. 两阶段路径的返回结果与旧的 MultiTraversalQuery 逐字一致（同一份数据上跑两遍对比）；
//  3. 完整管道只在持有该 cid 的那个库上执行 —— 用 mongo profiling 逐库数 aggregate 命令。
func TestMultiTraversalQueryByCidOnEthHashAgainstMongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("set MONGO_TEST_URI (scratch mongod, never a production/cold-storage node) to run this integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect %v: %v", uri, err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping %v: %v", uri, err)
	}

	// 三个「库」：cid 真正所在的库是 lb_cold_02；lb_cold_01 / lb_cold_03 里只有别的 cid。
	const (
		libOther = "ethhashprobe_lb_cold_01"
		libOwner = "ethhashprobe_lb_cold_02"
		libEmpty = "ethhashprobe_lb_cold_03"
	)
	libs := []string{libOther, libOwner, libEmpty}

	realCid := "bafy2bzacedethhashmessagecid"
	absentCid := "bafy2bzacedethhashabsentcid"
	ownerHash := "0x9c2a1f5e7b3d4c6a8e0f1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60"

	defer func() {
		for _, lib := range libs {
			_ = client.Database(lib).Drop(context.Background())
		}
	}()

	// 逐库准备数据 + 生产同款索引：EthHash 上是 Cid_1_Epoch_1（sparse 复合索引），
	// 没有 Cid 单键索引 —— 覆盖性依赖「复合索引前缀可服务 Cid 等值查询」。
	for _, lib := range libs {
		db := client.Database(lib)
		if err := db.RunCommand(ctx, bson.D{{Key: "profile", Value: 2}}).Err(); err != nil {
			t.Fatalf("enable profiling on %v: %v", lib, err)
		}

		if _, err := db.Collection("EthHash").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "Cid", Value: 1}, {Key: "Epoch", Value: 1}},
			Options: options.Index().SetName("Cid_1_Epoch_1").SetSparse(true),
		}); err != nil {
			t.Fatalf("create index on %v.EthHash: %v", lib, err)
		}
	}

	// 非 owner 库：放一条别的 cid，保证集合非空（探测 miss 不是因为集合是空的）
	for _, lib := range []string{libOther, libEmpty} {
		if _, err := client.Database(lib).Collection("EthHash").InsertOne(ctx, bson.M{
			"_id": "0xdeadbeef", "Cid": "bafy2bzacedothermessagecid", "Epoch": 1,
		}); err != nil {
			t.Fatalf("insert non-owner row on %v: %v", lib, err)
		}
	}

	// owner 库：eth hash 存在 _id 上，Cid 是消息 cid（与生产落库形态一致）
	if _, err := client.Database(libOwner).Collection("EthHash").InsertOne(ctx, bson.M{
		"_id": ownerHash, "Cid": realCid, "Epoch": 100,
	}); err != nil {
		t.Fatalf("insert owner row: %v", err)
	}

	countLists := make([]CountUtil, 0, len(libs))
	for _, lib := range libs {
		db := client.Database(lib)
		countLists = append(countLists, CountUtil{
			Cols:  common.Collections{DB: db, Cols: []*mongo.Collection{db.Collection("EthHash")}},
			DType: smodel.Cold,
		})
	}

	newPipe := func(cid string) interface{} {
		pipe, err := util.Parse(model.Ctx{Cid: cid}, string(monitor.GetHashByMessageCidAggregator()))
		if err != nil {
			t.Fatalf("parse production EthHash pipeline: %v", err)
		}
		return pipe
	}

	// ---- 1. 探测的真实执行计划：覆盖索引，不 FETCH 文档 -------------------------------
	t.Run("probe_plan_is_projection_covered_on_production_index_shape", func(t *testing.T) {
		var explain struct {
			QueryPlanner struct {
				WinningPlan bson.M `bson:"winningPlan"`
			} `bson:"queryPlanner"`
			ExecutionStats struct {
				TotalKeysExamined int64  `bson:"totalKeysExamined"`
				TotalDocsExamined int64  `bson:"totalDocsExamined"`
				ExecutionStages   bson.M `bson:"executionStages"`
			} `bson:"executionStats"`
		}

		// 与 mongoCidProbe 发出的命令逐字一致：find + filter + projection + limit(1)
		err := client.Database(libOwner).RunCommand(ctx, bson.D{
			{Key: "explain", Value: bson.D{
				{Key: "find", Value: "EthHash"},
				{Key: "filter", Value: cidProbeFilter(EthHashCidProbeFields[0], realCid)},
				{Key: "projection", Value: cidProbeProjection()},
				{Key: "limit", Value: 1},
			}},
			{Key: "verbosity", Value: "executionStats"},
		}).Decode(&explain)
		if err != nil {
			t.Fatalf("explain probe: %v", err)
		}

		plan, err := json.Marshal(explain.ExecutionStats.ExecutionStages)
		if err != nil {
			t.Fatalf("marshal plan: %v", err)
		}
		winning, err := json.Marshal(explain.QueryPlanner.WinningPlan)
		if err != nil {
			t.Fatalf("marshal winning plan: %v", err)
		}
		t.Logf("probe winningPlan: %s", winning)
		t.Logf("probe plan: %s", plan)
		t.Logf("keysExamined=%d docsExamined=%d", explain.ExecutionStats.TotalKeysExamined, explain.ExecutionStats.TotalDocsExamined)

		// 必须走 Cid_1_Epoch_1（复合前缀），否则覆盖性结论不成立
		if !strings.Contains(string(winning), "Cid_1_Epoch_1") {
			t.Fatalf("probe did not use the production index Cid_1_Epoch_1: %s", winning)
		}
		if explain.ExecutionStats.TotalDocsExamined != 0 {
			t.Fatalf("probe fetched documents: docsExamined=%d, plan=%s", explain.ExecutionStats.TotalDocsExamined, plan)
		}
		if explain.ExecutionStats.TotalKeysExamined != 1 {
			t.Fatalf("probe keysExamined=%d, want 1, plan=%s", explain.ExecutionStats.TotalKeysExamined, plan)
		}
		if strings.Contains(string(plan), "FETCH") {
			t.Fatalf("probe plan contains a FETCH stage, coverage broken: %s", plan)
		}
	})

	// ---- 1b. 旧路径的成本画像：逐库一次完整管道的真实计划 ---------------------------
	//
	// 用来核对「改造前后各碰几块盘、几次寻道」：探测是覆盖索引 seek；旧管道在
	// 未命中库上也是 IXSCAN（keysExamined=0，不 FETCH），只有 owner 库会 FETCH 一次
	// 文档（$project 要 $_id）。这两个数字决定了这次改造到底省下了什么。
	t.Run("legacy_pipeline_seek_profile", func(t *testing.T) {
		var explain struct {
			ExecutionStats struct {
				TotalKeysExamined int64  `bson:"totalKeysExamined"`
				TotalDocsExamined int64  `bson:"totalDocsExamined"`
				ExecutionStages   bson.M `bson:"executionStages"`
			} `bson:"executionStats"`
		}

		explainPipeline := func(lib string) (string, int64, int64, int64) {
			t.Helper()

			pipe := newPipe(realCid)

			// 与 MultiTraversalQuery 在冷库上发出的命令同形：aggregate + pipeline
			if err := client.Database(lib).RunCommand(ctx, bson.D{
				{Key: "explain", Value: bson.D{
					{Key: "aggregate", Value: "EthHash"},
					{Key: "pipeline", Value: pipe},
					{Key: "cursor", Value: bson.M{}},
				}},
				{Key: "verbosity", Value: "executionStats"},
			}).Decode(&explain); err != nil {
				t.Fatalf("explain aggregate on %v: %v", lib, err)
			}

			plan, err := json.Marshal(explain.ExecutionStats.ExecutionStages)
			if err != nil {
				t.Fatalf("marshal plan: %v", err)
			}

			return string(plan), explain.ExecutionStats.TotalKeysExamined,
				explain.ExecutionStats.TotalDocsExamined, countIndexSeeks(explain.ExecutionStats.ExecutionStages)
		}

		missPlan, missKeys, missDocs, missSeeks := explainPipeline(libOther)
		t.Logf("legacy pipeline on a NON-owner library: keysExamined=%d docsExamined=%d indexSeeks=%d plan=%s",
			missKeys, missDocs, missSeeks, missPlan)

		hitPlan, hitKeys, hitDocs, hitSeeks := explainPipeline(libOwner)
		t.Logf("legacy pipeline on the OWNER library: keysExamined=%d docsExamined=%d indexSeeks=%d plan=%s",
			hitKeys, hitDocs, hitSeeks, hitPlan)

		// 未命中库：走 Cid_1_Epoch_1 的 IXSCAN，一条文档都不读（所以旧路径的固定成本是
		// 「每库一次索引 seek」，不是「每库一次索引 seek + 一次文档读」）
		if missKeys != 0 || missDocs != 0 {
			t.Fatalf("legacy pipeline on a non-owner library examined keys=%d docs=%d, want 0/0", missKeys, missDocs)
		}
		if !strings.Contains(missPlan, "Cid_1_Epoch_1") {
			t.Fatalf("legacy pipeline on a non-owner library did not use Cid_1_Epoch_1: %s", missPlan)
		}

		// 命中库：$project 要 $_id ⇒ 非覆盖，必须 FETCH 一次文档
		if hitKeys != 1 || hitDocs != 1 {
			t.Fatalf("legacy pipeline on the owner library examined keys=%d docs=%d, want 1/1", hitKeys, hitDocs)
		}
		if !strings.Contains(hitPlan, "FETCH") {
			t.Fatalf("legacy pipeline on the owner library should FETCH the document ($project needs $_id): %s", hitPlan)
		}
	})

	// ---- 2. 结果与旧路径一致 + 管道只在 owner 库执行 ---------------------------------
	t.Run("result_matches_legacy_path_and_pipeline_runs_only_on_owner", func(t *testing.T) {
		pipe := newPipe(realCid)

		// 旧路径：3 个库各跑一次完整管道（这就是被省掉的固定成本）
		legacySnapshot := ethHashProfileSnapshot(ctx, t, client, libs)

		legacy, err := MultiTraversalQuery(ctx, pipe, countLists, "EthHash")
		if err != nil {
			t.Fatalf("legacy MultiTraversalQuery: %v", err)
		}

		for _, lib := range libs {
			if n := ethHashPipelineRunsSince(ctx, t, client, lib, legacySnapshot); n != 1 {
				t.Fatalf("legacy path ran the pipeline %d times on %v, want 1", n, lib)
			}
		}

		// 新路径：只有 owner 库跑完整管道；非 owner 库只付出一次覆盖索引探测
		newSnapshot := ethHashProfileSnapshot(ctx, t, client, libs)

		got, err := MultiTraversalQueryByCidOnTable(ctx, pipe, countLists, "EthHash", realCid, EthHashCidProbeFields)
		if err != nil {
			t.Fatalf("MultiTraversalQueryByCidOnTable: %v", err)
		}

		if n := ethHashPipelineRunsSince(ctx, t, client, libOwner, newSnapshot); n != 1 {
			t.Fatalf("two-phase path ran the pipeline %d times on the owner library, want 1", n)
		}
		for _, lib := range []string{libOther, libEmpty} {
			if n := ethHashPipelineRunsSince(ctx, t, client, lib, newSnapshot); n != 0 {
				t.Fatalf("two-phase path ran the pipeline %d times on non-owner library %v, want 0", n, lib)
			}
		}

		// 返回结果必须与旧路径逐字一致（Hash 来自文档 _id）
		if fmt.Sprint(got) != fmt.Sprint(legacy) {
			t.Fatalf("two-phase result %v != legacy result %v", got, legacy)
		}
		if len(got) != 1 {
			t.Fatalf("result = %v, want exactly 1 doc", got)
		}
		if gotHash := fmt.Sprint(got[0]["Hash"]); gotHash != ownerHash {
			t.Fatalf("Hash = %v, want %v", gotHash, ownerHash)
		}
	})

	// ---- 3. cid 不存在：探测全 miss ⇒ 一条完整管道都不跑，返回空且无错误 ----------
	t.Run("absent_cid_runs_no_pipeline", func(t *testing.T) {
		snapshot := ethHashProfileSnapshot(ctx, t, client, libs)

		res, err := MultiTraversalQueryByCidOnTable(ctx, newPipe(absentCid), countLists, "EthHash", absentCid, EthHashCidProbeFields)
		if err != nil {
			t.Fatalf("MultiTraversalQueryByCidOnTable: %v", err)
		}
		if len(res) != 0 {
			t.Fatalf("result = %v, want empty", res)
		}

		for _, lib := range libs {
			if n := ethHashPipelineRunsSince(ctx, t, client, lib, snapshot); n != 0 {
				t.Fatalf("pipeline ran %d times on %v for an absent cid", n, lib)
			}
		}
	})
}

// cursorStageOf 从 aggregate explain 的响应里取出 $cursor 阶段（里面才是
// queryPlanner/executionStats）。管道里只要有 $lookup/$unwind 这类非查询阶段，
// mongo 就返回 {stages: [{$cursor: ...}, {$lookup: ...}]} 形态。
func cursorStageOf(t *testing.T, raw bson.M) bson.M {
	t.Helper()

	stages, ok := raw["stages"].(bson.A)
	if !ok || len(stages) == 0 {
		t.Fatalf("explain response has no stages: %v", raw)
	}

	first, ok := stages[0].(bson.M)
	if !ok {
		t.Fatalf("explain stage 0 is not a document: %T", stages[0])
	}

	cursor, ok := first["$cursor"].(bson.M)
	if !ok {
		t.Fatalf("explain stage 0 is not $cursor: %v", first)
	}

	return cursor
}

// countIndexSeeks 在 explain 的 executionStages 树里累加 IXSCAN 阶段的 seeks 次数
// （mongo 自己统计的「索引 seek」次数，就是「每块盘几次寻道」里寻道那一半）。
func countIndexSeeks(stage bson.M) int64 {
	if stage == nil {
		return 0
	}

	var total int64

	if fmt.Sprint(stage["stage"]) == "IXSCAN" {
		if seeks, ok := stage["seeks"]; ok {
			switch v := seeks.(type) {
			case int32:
				total += int64(v)
			case int64:
				total += v
			case float64:
				total += int64(v)
			}
		}
	}

	for _, key := range []string{"inputStage", "child", "thenStage", "elseStage"} {
		if child, ok := stage[key].(bson.M); ok {
			total += countIndexSeeks(child)
		}
	}

	// SUBPLAN / OR 这类阶段把子阶段放在 inputStages 数组里（$or 的两个分支就是两个 IXSCAN）
	if children, ok := stage["inputStages"].(bson.A); ok {
		for _, child := range children {
			if childStage, ok := child.(bson.M); ok {
				total += countIndexSeeks(childStage)
			}
		}
	}

	return total
}

// ethHashAggregateCount 数某个库上「到目前为止」针对 EthHash 的 aggregate 命令次数
// （profiling level 2 会把每条命令写进 system.profile）。
func ethHashAggregateCount(ctx context.Context, t *testing.T, client *mongo.Client, lib string) int64 {
	t.Helper()

	n, err := client.Database(lib).Collection("system.profile").CountDocuments(ctx, bson.M{"command.aggregate": "EthHash"})
	if err != nil {
		t.Fatalf("count aggregate on %v: %v", lib, err)
	}

	return n
}

// ethHashProfileSnapshot 取各库当前的 EthHash aggregate 次数，用于之后算差值。
func ethHashProfileSnapshot(ctx context.Context, t *testing.T, client *mongo.Client, libs []string) map[string]int64 {
	t.Helper()

	snapshot := make(map[string]int64, len(libs))
	for _, lib := range libs {
		snapshot[lib] = ethHashAggregateCount(ctx, t, client, lib)
	}

	return snapshot
}

// ethHashPipelineRunsSince 返回某个库在快照之后执行了多少次完整管道。
func ethHashPipelineRunsSince(ctx context.Context, t *testing.T, client *mongo.Client, lib string, snapshot map[string]int64) int64 {
	t.Helper()

	return ethHashAggregateCount(ctx, t, client, lib) - snapshot[lib]
}
