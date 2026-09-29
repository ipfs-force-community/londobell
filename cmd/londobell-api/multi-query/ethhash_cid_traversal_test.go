package multiquery

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// ---------------------------------------------------------------------------
// /aggregators/hash_by_messagecid（EthHash）的两阶段改造：形态、与生产管道的一致性、
// 以及「返回结果不变」的编排级证明。
//
// 不连库：探测/聚合用 cid_traversal_test.go 里的 fakeProbe / fakeAgg 注入。
// 真库上的执行计划与新旧路径结果对比见 ethhash_cid_traversal_mongo_test.go。
// ---------------------------------------------------------------------------

// 1. 探测形态：EthHash 上只探 Cid 一个键，且保持覆盖索引形态。
func TestEthHashProbeShapeIsSingleKeyCoveredIndex(t *testing.T) {
	cid := "bafy2bzacedexamplecid"

	// EthHash 只有一种落库形态（管道只 $match Cid）⇒ 只允许一个探测键。
	// 多探一次 SignedCid 在 EthHash 上只有成本没有收益（11 个库各多一次冷盘 seek）。
	if !reflect.DeepEqual(EthHashCidProbeFields, []string{"Cid"}) {
		t.Fatalf("EthHash probe fields must be exactly [Cid], got %v", EthHashCidProbeFields)
	}

	// 单键等值：不能出现 $or（$or 会把计划退化成 SUBPLAN + FETCH）
	filter := cidProbeFilter(EthHashCidProbeFields[0], cid)
	if !reflect.DeepEqual(filter, bson.D{{Key: "Cid", Value: cid}}) {
		t.Fatalf("EthHash probe filter must be a single-key equality, got %v", filter)
	}
	if len(filter) != 1 {
		t.Fatalf("EthHash probe filter must have exactly 1 key, got %v", filter)
	}

	// 覆盖投影：正好 {_id:0, Cid:1}（EthHash 上的 Cid_1_Epoch_1 含 Cid，可覆盖）
	proj := cidProbeProjection()
	if !reflect.DeepEqual(proj, bson.D{{Key: "_id", Value: 0}, {Key: "Cid", Value: 1}}) {
		t.Fatalf("probe projection must be {_id:0, Cid:1}, got %v", proj)
	}
	if len(proj) != 2 {
		t.Fatalf("probe projection must project exactly 2 fields, got %v", proj)
	}
}

// 2. 与生产管道的一致性：探测键必须与管道 $match 逐字一致，且管道投影确实要文档 _id。
//
// 这条测试是「返回结果不变」的地基：
//   - 探测键 ⊆ 管道 $match ⇒ 不会出现「管道有结果但探测没命中」（漏查）；
//   - 这里两者相等 ⇒ 也不会出现「探测命中但管道必然为空」的形态（EthHash 上没有
//     IsBlock 这类会被管道再过滤掉的额外条件）；
//   - 管道 $project 里出现 Hash:"$_id" ⇒ 命中库必须 FETCH 文档 ⇒ 非覆盖，
//     只能用在「已经定位到库」之后，这正是拆两阶段的原因。
func TestEthHashPipelineMatchIsExactlyTheProbeFilter(t *testing.T) {
	cid := "bafy2bzacedexamplecid"

	pipe, err := util.Parse(model.Ctx{Cid: cid}, string(monitor.GetHashByMessageCidAggregator()))
	if err != nil {
		t.Fatalf("parse production EthHash pipeline: %v", err)
	}

	stages, ok := canonicalize(pipe).([]interface{})
	if !ok {
		t.Fatalf("EthHash pipeline must be an array of stages, got %T", canonicalize(pipe))
	}
	if len(stages) != 2 {
		t.Fatalf("EthHash pipeline must be [$match, $project], got %v", stages)
	}

	match, ok := stages[0].(map[string]interface{})["$match"].(map[string]interface{})
	if !ok {
		t.Fatalf("EthHash pipeline stage 0 must be $match, got %v", stages[0])
	}

	probeFilter, ok := canonicalize(cidProbeFilter(EthHashCidProbeFields[0], cid)).(map[string]interface{})
	if !ok {
		t.Fatalf("probe filter must be a single-key document, got %v", cidProbeFilter(EthHashCidProbeFields[0], cid))
	}

	if !reflect.DeepEqual(match, probeFilter) {
		t.Fatalf("EthHash pipeline $match %v != probe filter %v: 探测键必须与管道等值条件一致", match, probeFilter)
	}

	project, ok := stages[1].(map[string]interface{})["$project"].(map[string]interface{})
	if !ok {
		t.Fatalf("EthHash pipeline stage 1 must be $project, got %v", stages[1])
	}
	if _, hasHash := project["Hash"]; !hasHash {
		t.Fatalf("EthHash pipeline must keep projecting Hash from the document _id, got %v", project)
	}
	if got := fmt.Sprint(project["_id"]); got != "0" {
		t.Fatalf("EthHash pipeline must exclude _id from the raw document (Hash:\"$_id\"), got %v", project)
	}
}

// canonicalize 把 otto 导出的 bson.D/bson.M/bson.A 等结构统一成 map[string]interface{} /
// []interface{}，便于在测试里按字段路径断言（不依赖驱动对 interface{} 的默认解码类型）。
func canonicalize(v interface{}) interface{} {
	switch t := v.(type) {
	case bson.D:
		m := make(map[string]interface{}, len(t))
		for _, e := range t {
			m[e.Key] = canonicalize(e.Value)
		}
		return m
	case bson.M:
		m := make(map[string]interface{}, len(t))
		for k, val := range t {
			m[k] = canonicalize(val)
		}
		return m
	case bson.A:
		out := make([]interface{}, 0, len(t))
		for _, e := range t {
			out = append(out, canonicalize(e))
		}
		return out
	case []bson.D:
		out := make([]interface{}, 0, len(t))
		for _, e := range t {
			out = append(out, canonicalize(e))
		}
		return out
	case []bson.M:
		out := make([]interface{}, 0, len(t))
		for _, e := range t {
			out = append(out, canonicalize(e))
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(t))
		for _, e := range t {
			out = append(out, canonicalize(e))
		}
		return out
	default:
		return v
	}
}

// 3. 编排：只在命中库跑完整管道，返回结果与「直接在该库跑管道」逐字一致。
func TestTraversalQueryByCidOnEthHash(t *testing.T) {
	ethLib := func(name string, dtype smodel.DType) dbSpec {
		return dbSpec{name: name, dtype: dtype, cols: []string{"EthHash", "Message"}}
	}

	// 生产上 cid 唯一 ⇒ 只有一个库持有它；这里用真实的管道输出形态
	// （Hash 来自文档 _id）来证明返回值没被两阶段改写。
	ownerResult := []bson.M{{"Hash": "0x9c2a1f5e7b3d4c6a8e0f1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60"}}

	cases := []struct {
		name string

		dbs []dbSpec

		probeHits  map[string]map[string]bool
		probeErrs  map[string]error
		probeDelay map[string]time.Duration

		aggResults map[string][]bson.M
		aggErrs    map[string]error

		wantAggCalls    []string
		wantProbeFields map[string][]string
		wantNoProbe     []string

		wantResult []bson.M
		wantErr    string
	}{
		{
			// 主场景：11 个冷库里只有 cold_07 持有该 cid ⇒ 只有 cold_07 跑完整管道，
			// 其余库只付出一次覆盖索引探测。
			name: "only_the_owning_library_runs_the_pipeline",
			dbs: []dbSpec{
				ethLib("cold_01", smodel.Cold),
				ethLib("cold_07", smodel.Cold),
				ethLib("cold_11", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_07.EthHash": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"cold_07.EthHash": ownerResult,
			},
			wantAggCalls: []string{"cold_07.EthHash"},
			// 每个库只探 Cid 一次：EthHash 上没有 SignedCid 形态，不该出现第二个探测键
			wantProbeFields: map[string][]string{
				"cold_01.EthHash": {"Cid"},
				"cold_07.EthHash": {"Cid"},
				"cold_11.EthHash": {"Cid"},
			},
			wantResult: ownerResult,
		},
		{
			// 热点库（tmp/formal）命中即返回，冷库连探测都不发
			name: "hot_library_hit_short_circuits_cold",
			dbs: []dbSpec{
				ethLib("tmp_a", smodel.Tmp),
				ethLib("cold_01", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"tmp_a.EthHash": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"tmp_a.EthHash": ownerResult,
			},
			wantAggCalls:    []string{"tmp_a.EthHash"},
			wantProbeFields: map[string][]string{"tmp_a.EthHash": {"Cid"}},
			wantNoProbe:     []string{"cold_01.EthHash"},
			wantResult:      ownerResult,
		},
		{
			// 探测命中但管道为空（理论上的假命中）⇒ 继续试下一个候选库，
			// 保证不出现「探测命中却查不到」
			name: "hit_but_empty_pipeline_falls_through",
			dbs: []dbSpec{
				ethLib("cold_01", smodel.Cold),
				ethLib("cold_02", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_01.EthHash": {"Cid": true},
				"cold_02.EthHash": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"cold_02.EthHash": ownerResult,
			},
			wantAggCalls: []string{"cold_01.EthHash", "cold_02.EthHash"},
			wantResult:   ownerResult,
		},
		{
			// cid 不存在：一个完整管道都不跑，返回空（非 nil）且无错误 ——
			// 调用方（GetEthHashByCid）据此回落到「用 cid 推导 eth hash」的分支
			name: "absent_cid_runs_no_pipeline",
			dbs: []dbSpec{
				ethLib("cold_01", smodel.Cold),
				ethLib("cold_02", smodel.Cold),
			},
			wantAggCalls: []string{},
			wantProbeFields: map[string][]string{
				"cold_01.EthHash": {"Cid"},
				"cold_02.EthHash": {"Cid"},
			},
			wantResult: []bson.M{},
		},
		{
			// 探测出错必须上报（错误 ≠ 未命中），不能在别的库上瞎跑管道
			name: "probe_error_is_reported",
			dbs: []dbSpec{
				ethLib("cold_01", smodel.Cold),
				ethLib("cold_02", smodel.Cold),
			},
			probeErrs: map[string]error{
				"cold_02.EthHash": fmt.Errorf("no reachable servers"),
			},
			wantAggCalls: []string{},
			wantErr:      "probe cold_02.EthHash.Cid",
		},
		{
			// 命中库的管道报错必须上报，不能被当成「这个库里没有」
			name: "aggregate_error_is_reported",
			dbs: []dbSpec{
				ethLib("cold_01", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_01.EthHash": {"Cid": true},
			},
			aggErrs: map[string]error{
				"cold_01.EthHash": fmt.Errorf("exceeded time limit"),
			},
			wantAggCalls: []string{"cold_01.EthHash"},
			wantErr:      "aggregate cold_01.EthHash",
		},
		{
			// 冷库并发探测的完成顺序不能影响结果（命中库唯一，但要保证编排稳定）
			name: "concurrent_probe_result_is_deterministic",
			dbs: []dbSpec{
				ethLib("cold_01", smodel.Cold),
				ethLib("cold_02", smodel.Cold),
				ethLib("cold_03", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_03.EthHash": {"Cid": true},
			},
			probeDelay: map[string]time.Duration{
				"cold_01.EthHash": 30 * time.Millisecond,
				"cold_03.EthHash": 1 * time.Millisecond,
			},
			aggResults: map[string][]bson.M{
				"cold_03.EthHash": ownerResult,
			},
			wantAggCalls: []string{"cold_03.EthHash"},
			wantResult:   ownerResult,
		},
		{
			// 没有 EthHash 集合的库直接跳过（与 MultiTraversalQuery 一致，不报错）
			name: "libraries_without_the_table_are_skipped",
			dbs: []dbSpec{
				{name: "cold_01", dtype: smodel.Cold, cols: []string{"Message"}},
				ethLib("cold_02", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_02.EthHash": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"cold_02.EthHash": ownerResult,
			},
			wantAggCalls: []string{"cold_02.EthHash"},
			wantNoProbe:  []string{"cold_01.EthHash"},
			wantResult:   ownerResult,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			countLists := buildCountLists(t, tc.dbs)

			probe := &fakeProbe{hits: tc.probeHits, errs: tc.probeErrs, delay: tc.probeDelay}
			agg := &fakeAgg{results: tc.aggResults, errs: tc.aggErrs}

			// 与 pool-monitor/hash_by_messagecid.js 同形的管道：$project 要文档 _id
			pipe := []bson.M{
				{"$match": bson.M{"Cid": "x"}},
				{"$project": bson.M{"_id": 0, "Hash": "$_id"}},
			}
			cid := "bafy2bzacedexamplecid"

			res, err := traversalQueryByCid(context.Background(), probe.probe, agg.agg, pipe,
				countLists, "EthHash", cid, EthHashCidProbeFields)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil (res=%v)", tc.wantErr, res)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := agg.called(); !reflect.DeepEqual(got, tc.wantAggCalls) {
				t.Fatalf("aggregate calls = %v, want %v", got, tc.wantAggCalls)
			}

			// 「返回结果不变」：结果必须与在命中库上直接跑管道拿到的原始输出逐字一致
			// （Hash 来自 $_id 的形态不能被动过）
			if tc.wantErr == "" && !reflect.DeepEqual(res, tc.wantResult) {
				t.Fatalf("result = %v, want %v (must equal the raw pipeline output)", res, tc.wantResult)
			}

			for desc, wantFields := range tc.wantProbeFields {
				if got := probe.fieldsFor(desc); !reflect.DeepEqual(got, wantFields) {
					t.Fatalf("probe fields for %v = %v, want %v", desc, got, wantFields)
				}
			}
			for _, desc := range tc.wantNoProbe {
				if got := probe.fieldsFor(desc); len(got) != 0 {
					t.Fatalf("library %v must not be probed, got %v", desc, got)
				}
			}

			// 管道与表名必须原样透传
			for i := range agg.pipes {
				if !reflect.DeepEqual(agg.pipes[i], pipe) {
					t.Fatalf("pipe #%d was modified: %v", i, agg.pipes[i])
				}
				if agg.tables[i] != "EthHash" {
					t.Fatalf("tableName #%d = %v, want EthHash", i, agg.tables[i])
				}
			}
		})
	}
}

// 4. 新旧路径等价（编排级）：同一个 cid、同一个管道，两阶段路径与「按 MultiTraversalQuery
// 的语义（热点库优先、冷库并发取第一个非空结果）遍历」得到的结果必须一致。
//
// MultiTraversalQuery 本身要真库（它直接操作 *mongo.Collection），所以这里用一个
// 只依赖同一组 fake 的参照实现来对齐语义：cid 唯一 ⇒ 结果 = 持有该 cid 的那个库的管道输出。
func TestTraversalQueryByCidOnEthHashMatchesLegacySemantics(t *testing.T) {
	countLists := buildCountLists(t, []dbSpec{
		{name: "tmp_a", dtype: smodel.Tmp, cols: []string{"EthHash"}},
		{name: "cold_01", dtype: smodel.Cold, cols: []string{"EthHash"}},
		{name: "cold_02", dtype: smodel.Cold, cols: []string{"EthHash"}},
	})

	cid := "bafy2bzacedexamplecid"
	// 旧路径在冷库上并发跑、取第一个非空结果；cid 唯一 ⇒ 只有 owning 库非空。
	legacyResults := map[string][]bson.M{
		"tmp_a.EthHash":   {},
		"cold_01.EthHash": {},
		"cold_02.EthHash": {{"Hash": "0xowner"}},
	}

	pipe := []bson.M{{"$match": bson.M{"Cid": cid}}, {"$project": bson.M{"_id": 0, "Hash": "$_id"}}}

	probe := &fakeProbe{hits: map[string]map[string]bool{"cold_02.EthHash": {"Cid": true}}}
	agg := &fakeAgg{results: legacyResults}

	got, err := traversalQueryByCid(context.Background(), probe.probe, agg.agg, pipe,
		countLists, "EthHash", cid, EthHashCidProbeFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 参照语义：旧路径返回 owning 库的原始管道输出
	want := legacyResults["cold_02.EthHash"]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("two-phase result = %v, legacy semantics = %v", got, want)
	}

	// 新路径在 owning 库之外一个管道都不跑（这就是省下来的成本）
	if calls := agg.called(); !reflect.DeepEqual(calls, []string{"cold_02.EthHash"}) {
		t.Fatalf("aggregate calls = %v, want only [cold_02.EthHash]", calls)
	}
}
