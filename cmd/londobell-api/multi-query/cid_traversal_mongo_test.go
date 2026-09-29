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

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// 这个集成测试需要一个「一次性」的 mongod（本地 scratch 实例即可，绝不要指向生产/冷库）：
//
//	MONGO_TEST_URI=mongodb://127.0.0.1:27099 go test ./cmd/londobell-api/multi-query/ \
//	    -run TestMultiTraversalQueryByCidAgainstMongo -v
//
// 没设置 MONGO_TEST_URI 时整体跳过，保证 `go test ./...` 在没有 mongo 的环境上依然全绿。
//
// 它验证三件在假实现（cid_traversal_test.go）里验不到的事：
//  1. 探测的真实执行计划：keysExamined=1 / docsExamined=0，且计划里没有 FETCH
//     （= 覆盖索引形态成立，不读文档页）；
//  2. 假命中（探测命中但 $match 的 IsBlock 把它过滤掉）时，完整管道确实只在
//     「该 cid 真正所在的那个库」上执行过 —— 用 mongo 的 profiling 逐库数 aggregate 命令；
//  3. SignedCid-only 落库形态能定位到库并返回与原来一致的文档。
func TestMultiTraversalQueryByCidAgainstMongo(t *testing.T) {
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

	// 三个「库」：cid 真正所在的库是 lb_cold_02；lb_cold_01 上放一条 IsBlock=false 的假命中；
	// lb_cold_03 只有索引没有数据。
	const (
		libFalseHit = "cidprobe_lb_cold_01"
		libReal     = "cidprobe_lb_cold_02"
		libEmpty    = "cidprobe_lb_cold_03"
	)
	libs := []string{libFalseHit, libReal, libEmpty}

	realCid := "bafy2bzacedrealmessagecid"
	fakeCid := "bafy2bzacedfalsehitnotexist"
	signedCid := "bafy2bzacedoutsidesignedcid"
	innerCid := "bafy2bzacedinnerunsignedcid"

	defer func() {
		for _, lib := range libs {
			_ = client.Database(lib).Drop(context.Background())
		}
	}()

	// 逐库准备数据 + 生产同款索引（Cid_1 / SignedCid_1）
	for _, lib := range libs {
		db := client.Database(lib)
		if err := db.RunCommand(ctx, bson.D{{Key: "profile", Value: 2}}).Err(); err != nil {
			t.Fatalf("enable profiling on %v: %v", lib, err)
		}

		traceCol := db.Collection("ExecTrace")
		if _, err := traceCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
			{Keys: bson.D{{Key: "Cid", Value: 1}}, Options: options.Index().SetName("Cid_1")},
			{Keys: bson.D{{Key: "SignedCid", Value: 1}}, Options: options.Index().SetName("SignedCid_1")},
		}); err != nil {
			t.Fatalf("create index on %v.ExecTrace: %v", lib, err)
		}
	}

	// 假命中库：同 cid 的行 IsBlock=false ⇒ 探测命中，但管道里的 {$and:[{IsBlock:true},...]} 会过滤掉它
	if _, err := client.Database(libFalseHit).Collection("ExecTrace").InsertOne(ctx, bson.M{
		"Cid": realCid, "IsBlock": false, "Epoch": 1,
	}); err != nil {
		t.Fatalf("insert false-hit row: %v", err)
	}

	// 真库：一条正常块内消息（Cid）+ 一条带签名的消息（SignedCid 落库形态）+ 对应 Message
	if _, err := client.Database(libReal).Collection("ExecTrace").InsertMany(ctx, []interface{}{
		bson.M{
			"Cid": realCid, "IsBlock": true, "Epoch": 100,
			"MsgRct": bson.M{"ExitCode": 0}, "Seq": []int64{0},
		},
		bson.M{
			"Cid": innerCid, "SignedCid": signedCid, "IsBlock": true, "Epoch": 101,
			"MsgRct": bson.M{"ExitCode": 0}, "Seq": []int64{0},
		},
	}); err != nil {
		t.Fatalf("insert trace rows: %v", err)
	}

	if _, err := client.Database(libReal).Collection("Message").InsertMany(ctx, []interface{}{
		// 未签名消息：SignedCid 落 null（管道的 $cond 正是靠 message.SignedCid == null 取 message._id）
		bson.M{"_id": realCid, "SignedCid": nil, "From": "f01000", "To": "f01001", "Value": "1", "Nonce": 1},
		bson.M{"_id": innerCid, "SignedCid": signedCid, "From": "f01000", "To": "f01002", "Value": "2", "Nonce": 2},
	}); err != nil {
		t.Fatalf("insert message rows: %v", err)
	}

	countLists := make([]CountUtil, 0, len(libs))
	for _, lib := range libs {
		db := client.Database(lib)
		countLists = append(countLists, CountUtil{
			Cols:  common.Collections{DB: db, Cols: []*mongo.Collection{db.Collection("ExecTrace"), db.Collection("Message")}},
			DType: smodel.Cold,
		})
	}

	// 与 pool-monitor/trace_for_message.js 同形态的最小管道（$match IsBlock + $or(Cid/SignedCid)
	// → $lookup Message → $unwind → $project）：后半段必须取文档，所以只能用在「已经定位到库」之后。
	const probeJsPipe = `
	[
	    {$match: {$and: [{IsBlock: true}, {$or: [{Cid: ctx.Cid}, {SignedCid: ctx.Cid}]}]}},
	    {$lookup: {from: "Message", localField: "Cid", foreignField: "_id", as: "message"}},
	    {$unwind: "$message"},
	    {$project: {Cid: {$cond: {if: {$eq: ["$message.SignedCid", null]}, then: "$message._id", else: "$message.SignedCid"}},
	                Epoch: "$Epoch", From: "$message.From", To: "$message.To"}}
	]`

	// ---- 1. 探测的真实执行计划：覆盖索引，不 FETCH 文档 -------------------------------
	t.Run("probe_plan_is_projection_covered", func(t *testing.T) {
		var explain struct {
			ExecutionStats struct {
				TotalKeysExamined int64  `bson:"totalKeysExamined"`
				TotalDocsExamined int64  `bson:"totalDocsExamined"`
				ExecutionStages   bson.M `bson:"executionStages"`
			} `bson:"executionStats"`
		}

		// 与 mongoCidProbe 发出的命令逐字一致：find + filter + projection + limit(1)
		err := client.Database(libReal).RunCommand(ctx, bson.D{
			{Key: "explain", Value: bson.D{
				{Key: "find", Value: "ExecTrace"},
				{Key: "filter", Value: cidProbeFilter("Cid", realCid)},
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
		t.Logf("probe plan: %s", plan)
		t.Logf("keysExamined=%d docsExamined=%d", explain.ExecutionStats.TotalKeysExamined, explain.ExecutionStats.TotalDocsExamined)

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

	// ---- 1b. 旧路径的成本画像：逐库一次完整管道在未命中库上的真实计划 ----------------
	//
	// 这条子测试回答「旧路径每库几次寻道」：ExecTrace 的管道 $match 是
	// {$and:[{IsBlock:true},{$or:[{Cid:X},{SignedCid:X}]}]} ⇒ 计划是 SUBPLAN 下挂两个
	// IXSCAN（Cid_1 / SignedCid_1），未命中库上每个分支各 seek 一次 = 每库 2 次寻道，
	// 11 个冷库就是 22 次 —— 这正是「22 次寻道/请求」的来源，而单键等值的 EthHash
	// 管道（hash_by_messagecid）每库只有 1 次寻道。
	t.Run("legacy_pipeline_seek_profile", func(t *testing.T) {
		pipe, err := util.Parse(map[string]interface{}{"Cid": fakeCid}, probeJsPipe)
		if err != nil {
			t.Fatalf("parse pipeline: %v", err)
		}

		// 与 MultiTraversalQuery 在冷库上发出的命令同形：aggregate + pipeline。
		// 管道里有 $lookup/$unwind，mongo 的 explain 因此返回 stages 形态
		// （stages[0].$cursor 里才是 queryPlanner/executionStats）。
		var raw bson.M
		if err := client.Database(libReal).RunCommand(ctx, bson.D{
			{Key: "explain", Value: bson.D{
				{Key: "aggregate", Value: "ExecTrace"},
				{Key: "pipeline", Value: pipe},
				{Key: "cursor", Value: bson.M{}},
			}},
			{Key: "verbosity", Value: "executionStats"},
		}).Decode(&raw); err != nil {
			t.Fatalf("explain aggregate: %v", err)
		}
		if ok, _ := raw["ok"].(float64); ok != 1 {
			t.Fatalf("explain aggregate rejected: %v", raw)
		}

		cursorStage := cursorStageOf(t, raw)

		queryPlanner, _ := cursorStage["queryPlanner"].(bson.M)
		plan, err := json.Marshal(queryPlanner["winningPlan"])
		if err != nil {
			t.Fatalf("marshal winning plan: %v", err)
		}

		stats, _ := cursorStage["executionStats"].(bson.M)
		keys, _ := stats["totalKeysExamined"].(int32)
		docs, _ := stats["totalDocsExamined"].(int32)
		execStages, _ := stats["executionStages"].(bson.M)
		seeks := countIndexSeeks(execStages)

		t.Logf("legacy ExecTrace pipeline on a NON-owner library: keysExamined=%d docsExamined=%d indexSeeks=%d winningPlan=%s",
			keys, docs, seeks, plan)

		// $or 的两个分支各 seek 一次（Cid_1 / SignedCid_1）：这就是 11 个冷库 = 22 次寻道的来源。
		// 对照：EthHash（hash_by_messagecid）的 $match 是单键等值，每库只有 1 次寻道 ——
		// 所以「22 次寻道」这个数字属于 ExecTrace，不属于 hash_by_messagecid。
		if seeks != 2 {
			t.Fatalf("legacy ExecTrace pipeline should seek once per $or branch (2), got %d: %s", seeks, plan)
		}
		if keys != 0 || docs != 0 {
			t.Fatalf("legacy ExecTrace pipeline on a non-owner library examined keys=%d docs=%d, want 0/0", keys, docs)
		}
	})

	// ---- 2. 假命中继续 + 管道只在命中库执行 -----------------------------------------
	t.Run("false_hit_falls_through_and_pipeline_runs_only_on_the_owning_library", func(t *testing.T) {
		snapshot := profileSnapshot(ctx, t, client, libs)

		pipe, err := util.Parse(map[string]interface{}{"Cid": realCid}, probeJsPipe)
		if err != nil {
			t.Fatalf("parse pipeline: %v", err)
		}

		res, err := MultiTraversalQueryByCid(ctx, pipe, countLists, "ExecTrace", realCid)
		if err != nil {
			t.Fatalf("MultiTraversalQueryByCid: %v", err)
		}

		if len(res) != 1 {
			t.Fatalf("result = %v, want exactly 1 doc", res)
		}
		if got := fmt.Sprint(res[0]["Cid"]); got != realCid {
			t.Fatalf("Cid = %v, want %v", got, realCid)
		}

		// 假命中库上完整管道跑过（探测命中才会跑），真库上完整管道跑过并出结果，
		// 空库上一条 aggregate 都不该有 —— 这就是「只在命中库跑管道」的机器证据。
		if n := pipelineRunsSince(ctx, t, client, libFalseHit, snapshot); n != 1 {
			t.Fatalf("pipeline ran %d times on false-hit library, want 1", n)
		}
		if n := pipelineRunsSince(ctx, t, client, libReal, snapshot); n != 1 {
			t.Fatalf("pipeline ran %d times on owning library, want 1", n)
		}
		if n := pipelineRunsSince(ctx, t, client, libEmpty, snapshot); n != 0 {
			t.Fatalf("pipeline ran %d times on non-hit library, want 0", n)
		}
	})

	// ---- 3. SignedCid-only 落库形态 -------------------------------------------------
	t.Run("signed_cid_only_row_is_found", func(t *testing.T) {
		snapshot := profileSnapshot(ctx, t, client, libs)

		pipe, err := util.Parse(map[string]interface{}{"Cid": signedCid}, probeJsPipe)
		if err != nil {
			t.Fatalf("parse pipeline: %v", err)
		}

		res, err := MultiTraversalQueryByCid(ctx, pipe, countLists, "ExecTrace", signedCid)
		if err != nil {
			t.Fatalf("MultiTraversalQueryByCid: %v", err)
		}

		if len(res) != 1 {
			t.Fatalf("result = %v, want exactly 1 doc", res)
		}
		if got := fmt.Sprint(res[0]["Cid"]); got != signedCid {
			t.Fatalf("Cid = %v, want %v (pipeline 投影优先取 message.SignedCid)", got, signedCid)
		}

		if n := pipelineRunsSince(ctx, t, client, libReal, snapshot); n != 1 {
			t.Fatalf("pipeline ran %d times on owning library, want 1", n)
		}
		if n := pipelineRunsSince(ctx, t, client, libFalseHit, snapshot); n != 0 {
			t.Fatalf("pipeline ran %d times on non-owning library, want 0", n)
		}
	})

	// ---- 4. cid 完全不存在：探测全 miss ⇒ 一条完整管道都不跑，返回空且无错误 --------
	t.Run("absent_cid_runs_no_pipeline", func(t *testing.T) {
		snapshot := profileSnapshot(ctx, t, client, libs)

		pipe, err := util.Parse(map[string]interface{}{"Cid": fakeCid}, probeJsPipe)
		if err != nil {
			t.Fatalf("parse pipeline: %v", err)
		}

		res, err := MultiTraversalQueryByCid(ctx, pipe, countLists, "ExecTrace", fakeCid)
		if err != nil {
			t.Fatalf("MultiTraversalQueryByCid: %v", err)
		}
		if len(res) != 0 {
			t.Fatalf("result = %v, want empty", res)
		}

		for _, lib := range libs {
			if n := pipelineRunsSince(ctx, t, client, lib, snapshot); n != 0 {
				t.Fatalf("pipeline ran %d times on %v for an absent cid", n, lib)
			}
		}
	})
}

// aggregateCount 数某个库上「到目前为止」针对 ExecTrace 的 aggregate 命令次数
// （profiling level 2 会把每条命令写进 system.profile）。
//
// system.profile 是服务端自建的 capped 集合、不允许客户端删记录，
// 所以这里用「查询前后取差值」而不是清空。
func aggregateCount(ctx context.Context, t *testing.T, client *mongo.Client, lib string) int64 {
	t.Helper()

	n, err := client.Database(lib).Collection("system.profile").CountDocuments(ctx, bson.M{"command.aggregate": "ExecTrace"})
	if err != nil {
		t.Fatalf("count aggregate on %v: %v", lib, err)
	}

	return n
}

// profileSnapshot 取各库当前的 aggregate 次数，用于之后算差值。
func profileSnapshot(ctx context.Context, t *testing.T, client *mongo.Client, libs []string) map[string]int64 {
	t.Helper()

	snapshot := make(map[string]int64, len(libs))
	for _, lib := range libs {
		snapshot[lib] = aggregateCount(ctx, t, client, lib)
	}

	return snapshot
}

// pipelineRunsSince 返回某个库在快照之后执行了多少次完整管道。
func pipelineRunsSince(ctx context.Context, t *testing.T, client *mongo.Client, lib string, snapshot map[string]int64) int64 {
	t.Helper()

	return aggregateCount(ctx, t, client, lib) - snapshot[lib]
}
