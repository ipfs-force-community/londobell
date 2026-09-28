package multiquery

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// ---------------------------------------------------------------------------
// 假库构造：只需要拿到非 nil 的 *mongo.Collection（用来做库/集合标识与错误信息），
// 不连真库。mongo.NewClient 只构造客户端、不做任何 I/O（拓扑监控在 Client.Connect
// 里才启动），所以这里既不会连库，也不会留后台 goroutine。
// ---------------------------------------------------------------------------

var (
	testClientOnce sync.Once
	testClientVal  *mongo.Client
	testClientErr  error
)

func testClient(t *testing.T) *mongo.Client {
	t.Helper()

	testClientOnce.Do(func() {
		testClientVal, testClientErr = mongo.NewClient(options.Client().ApplyURI("mongodb://127.0.0.1:1/unit-test"))
	})

	if testClientErr != nil {
		t.Fatalf("new fake mongo client: %v", testClientErr)
	}

	return testClientVal
}

type dbSpec struct {
	name  string
	dtype smodel.DType
	cols  []string
}

func buildCountLists(t *testing.T, specs []dbSpec) []CountUtil {
	t.Helper()

	countLists := make([]CountUtil, 0, len(specs))
	for _, spec := range specs {
		db := testClient(t).Database(spec.name)

		cols := make([]*mongo.Collection, 0, len(spec.cols))
		for _, colName := range spec.cols {
			cols = append(cols, db.Collection(colName))
		}

		countLists = append(countLists, CountUtil{
			Cols:  common.Collections{DB: db, Cols: cols},
			DType: spec.dtype,
		})
	}

	return countLists
}

// ---------------------------------------------------------------------------
// 注入用的假探测 / 假聚合：不碰网络，只记录调用，便于断言「探了哪些库、按什么顺序、
// 命中后在哪几个库上跑了完整管道」。
// ---------------------------------------------------------------------------

type fakeProbe struct {
	mu    sync.Mutex
	hits  map[string]map[string]bool // desc -> field -> 是否命中
	errs  map[string]error           // desc -> 探测错误
	calls []string                   // "desc.field"，按调用顺序
	delay map[string]time.Duration   // desc -> 人工延迟，用于验证并发结果的顺序还原
}

func (f *fakeProbe) probe(_ context.Context, col *mongo.Collection, field, _ string) (bool, error) {
	desc := columnDesc(col)

	f.mu.Lock()
	f.calls = append(f.calls, desc+"."+field)
	err := f.errs[desc]
	hit := f.hits[desc][field]
	delay := f.delay[desc]
	f.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	if err != nil {
		return false, err
	}

	return hit, nil
}

// fieldsFor 返回某个库上探测过的字段顺序（并发探测下，库内顺序才是确定的）。
func (f *fakeProbe) fieldsFor(desc string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	fields := make([]string, 0, len(CidProbeFields))
	for _, call := range f.calls {
		if strings.HasPrefix(call, desc+".") {
			fields = append(fields, strings.TrimPrefix(call, desc+"."))
		}
	}

	return fields
}

func (f *fakeProbe) descs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string{}, f.calls...)
}

type fakeAgg struct {
	mu      sync.Mutex
	results map[string][]bson.M // desc -> 管道结果
	errs    map[string]error    // desc -> 管道错误
	calls   []string            // desc，按调用顺序
	pipes   []interface{}       // 收到的 pipe（断言透传）
	tables  []string            // 收到的 tableName
}

func (f *fakeAgg) agg(_ context.Context, col *mongo.Collection, pipe interface{}, tableName string) ([]bson.M, error) {
	desc := columnDesc(col)

	f.mu.Lock()
	f.calls = append(f.calls, desc)
	f.pipes = append(f.pipes, pipe)
	f.tables = append(f.tables, tableName)
	err := f.errs[desc]
	res := f.results[desc]
	f.mu.Unlock()

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (f *fakeAgg) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string{}, f.calls...)
}

// ---------------------------------------------------------------------------
// 1. 探测形态：单键 + 覆盖索引投影（不连库的机器检查）
// ---------------------------------------------------------------------------

func TestCidProbeShapeIsSingleKeyCoveredIndex(t *testing.T) {
	cid := "bafy2bzacedexamplecid"

	// 单键：必须正好一条 {Cid: cid}，不能出现 $or（$or 会把计划变成 SUBPLAN + FETCH）
	filter := cidProbeFilter("Cid", cid)
	if !reflect.DeepEqual(filter, bson.D{{Key: "Cid", Value: cid}}) {
		t.Fatalf("probe filter must be a single-key equality, got %v", filter)
	}
	for _, elem := range filter {
		if strings.HasPrefix(elem.Key, "$") {
			t.Fatalf("probe filter must not contain operators, got %v", filter)
		}
	}

	// 投影：必须是 {_id:0, Cid:1}（Cid_1 索引覆盖，不 FETCH 文档）
	proj := cidProbeProjection()
	if !reflect.DeepEqual(proj, bson.D{{Key: "_id", Value: 0}, {Key: "Cid", Value: 1}}) {
		t.Fatalf("probe projection must be {_id:0, Cid:1}, got %v", proj)
	}
	if len(proj) != 2 {
		t.Fatalf("probe projection must project exactly 2 fields, got %v", proj)
	}

	// 探测字段顺序：先 Cid，未命中再 SignedCid
	if !reflect.DeepEqual(CidProbeFields, []string{"Cid", "SignedCid"}) {
		t.Fatalf("probe fields order must be [Cid SignedCid], got %v", CidProbeFields)
	}
}

// ---------------------------------------------------------------------------
// 2. 候选选择：表过滤 + 顺序
// ---------------------------------------------------------------------------

func TestCidCandidatesSkipLibrariesWithoutTable(t *testing.T) {
	countLists := buildCountLists(t, []dbSpec{
		{name: "tmp_a", dtype: smodel.Tmp, cols: []string{"ExecTrace", "Message"}},
		{name: "cold_01", dtype: smodel.Cold, cols: []string{"Message", "ExecTrace"}},
		{name: "cold_02", dtype: smodel.Cold, cols: []string{"Message"}}, // 没有目标表 ⇒ 跳过
		{name: "formal_a", dtype: smodel.Formal, cols: []string{"ExecTrace"}},
	})

	candidates := cidCandidates(countLists, "ExecTrace")

	wantDescs := []string{"tmp_a.ExecTrace", "cold_01.ExecTrace", "formal_a.ExecTrace"}
	gotDescs := make([]string, 0, len(candidates))
	for i, cand := range candidates {
		gotDescs = append(gotDescs, cand.desc)
		if cand.order != i {
			t.Fatalf("candidate %d order = %d, want %d", i, cand.order, i)
		}
	}

	if !reflect.DeepEqual(gotDescs, wantDescs) {
		t.Fatalf("candidates = %v, want %v", gotDescs, wantDescs)
	}
}

// ---------------------------------------------------------------------------
// 3. 主流程：表驱动覆盖「热点优先 / 单键探测 / 假命中继续 / 全空 / 错误上报」
// ---------------------------------------------------------------------------

type traversalCase struct {
	name string

	dbs []dbSpec

	probeHits  map[string]map[string]bool
	probeErrs  map[string]error
	probeDelay map[string]time.Duration

	aggResults map[string][]bson.M
	aggErrs    map[string]error

	// 期望：命中库上跑完整管道的顺序
	wantAggCalls []string
	// 期望：各库探测过的字段顺序（冷库是并发探测，只断言库内顺序）
	wantProbeFields map[string][]string
	// 期望：完全不该被探测的库（用于验证热点库命中后不再探冷库）
	wantNoProbe []string

	wantResultCount int
	wantErr         string
}

func TestTraversalQueryByCid(t *testing.T) {
	db1 := func(name string, dtype smodel.DType) dbSpec {
		return dbSpec{name: name, dtype: dtype, cols: []string{"ExecTrace", "Message"}}
	}

	cases := []traversalCase{
		{
			// formal/tmp 命中就直接返回，冷库连探测都不发（与 MultiTraversalQuery 的
			// 「优先查 tmp/formal，未查到再并发查冷库」一致）
			name: "hot_library_hit_short_circuits_cold",
			dbs: []dbSpec{
				db1("tmp_a", smodel.Tmp),
				db1("formal_a", smodel.Formal),
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"tmp_a.ExecTrace": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"tmp_a.ExecTrace": {{"Cid": "x", "IsBlock": true}},
			},
			wantAggCalls: []string{"tmp_a.ExecTrace"},
			wantProbeFields: map[string][]string{
				"tmp_a.ExecTrace": {"Cid"},
			},
			// 热点库一命中就返回：formal 与冷库连探测都不发
			wantNoProbe:     []string{"formal_a.ExecTrace", "cold_01.ExecTrace", "cold_02.ExecTrace"},
			wantResultCount: 1,
		},
		{
			// 热点库探测命中但管道为空 ⇒ 继续试下一个热点库（与原来的 priorityLists
			// 顺序循环一致：前一个库空了才看后一个）
			name: "hot_probe_hit_but_empty_falls_through_to_next_hot",
			dbs: []dbSpec{
				db1("tmp_a", smodel.Tmp),
				db1("formal_a", smodel.Formal),
				db1("cold_01", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"tmp_a.ExecTrace":    {"Cid": true},
				"formal_a.ExecTrace": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"formal_a.ExecTrace": {{"Cid": "h", "IsBlock": true}},
			},
			wantAggCalls: []string{"tmp_a.ExecTrace", "formal_a.ExecTrace"},
			wantProbeFields: map[string][]string{
				"tmp_a.ExecTrace":    {"Cid"},
				"formal_a.ExecTrace": {"Cid"},
			},
			wantNoProbe:     []string{"cold_01.ExecTrace"},
			wantResultCount: 1,
		},
		{
			// 热点库探测报错 ⇒ 立刻上报，后面的库不再触碰（与原来「热点库先跑、
			// 出错即返回」一致）
			name: "hot_probe_error_short_circuits",
			dbs: []dbSpec{
				db1("tmp_a", smodel.Tmp),
				db1("formal_a", smodel.Formal),
				db1("cold_01", smodel.Cold),
			},
			probeErrs: map[string]error{
				"tmp_a.ExecTrace": fmt.Errorf("no reachable servers"),
			},
			wantAggCalls: []string{},
			wantErr:      "probe tmp_a.ExecTrace.Cid",
			wantNoProbe:  []string{"formal_a.ExecTrace", "cold_01.ExecTrace"},
		},
		{
			// 热点库探测未命中 ⇒ 冷库并发探测；SignedCid 命中（Cid 未命中时才探 SignedCid）
			name: "signed_cid_only_hit_in_cold",
			dbs: []dbSpec{
				db1("tmp_a", smodel.Tmp),
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_02.ExecTrace": {"SignedCid": true},
			},
			aggResults: map[string][]bson.M{
				"cold_02.ExecTrace": {{"Cid": "y", "IsBlock": true}},
			},
			wantAggCalls: []string{"cold_02.ExecTrace"},
			wantProbeFields: map[string][]string{
				"tmp_a.ExecTrace":   {"Cid", "SignedCid"},
				"cold_01.ExecTrace": {"Cid", "SignedCid"},
				"cold_02.ExecTrace": {"Cid", "SignedCid"},
			},
			wantResultCount: 1,
		},
		{
			// 假命中：探测命中但管道为空（例如那一行 IsBlock=false），继续试下一个候选库
			name: "fake_hit_continues_to_next_candidate",
			dbs: []dbSpec{
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
				db1("cold_03", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_01.ExecTrace": {"Cid": true},
				"cold_03.ExecTrace": {"Cid": true},
			},
			aggResults: map[string][]bson.M{
				"cold_03.ExecTrace": {{"Cid": "z", "IsBlock": true}},
			},
			wantAggCalls: []string{"cold_01.ExecTrace", "cold_03.ExecTrace"},
			wantProbeFields: map[string][]string{
				"cold_01.ExecTrace": {"Cid"},
				"cold_02.ExecTrace": {"Cid", "SignedCid"},
				"cold_03.ExecTrace": {"Cid"},
			},
			wantResultCount: 1,
		},
		{
			// 全部命中库管道都为空 ⇒ 才返回空（无错误）
			name: "all_hits_empty_returns_empty",
			dbs: []dbSpec{
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_01.ExecTrace": {"Cid": true},
				"cold_02.ExecTrace": {"SignedCid": true},
			},
			wantAggCalls: []string{"cold_01.ExecTrace", "cold_02.ExecTrace"},
			wantProbeFields: map[string][]string{
				"cold_01.ExecTrace": {"Cid"},
				"cold_02.ExecTrace": {"Cid", "SignedCid"},
			},
			wantResultCount: 0,
		},
		{
			// 一个库都没探测命中 ⇒ 一个完整管道都不跑
			name: "no_probe_hit_runs_no_pipeline",
			dbs: []dbSpec{
				db1("tmp_a", smodel.Tmp),
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
			},
			wantAggCalls: []string{},
			wantProbeFields: map[string][]string{
				"tmp_a.ExecTrace":   {"Cid", "SignedCid"},
				"cold_01.ExecTrace": {"Cid", "SignedCid"},
				"cold_02.ExecTrace": {"Cid", "SignedCid"},
			},
			wantResultCount: 0,
		},
		{
			// 探测报错必须上报（错误 ≠ 未命中），且不能继续在别的库上瞎跑管道
			name: "probe_error_is_returned",
			dbs: []dbSpec{
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
			},
			probeErrs: map[string]error{
				"cold_02.ExecTrace": fmt.Errorf("connection refused"),
			},
			wantAggCalls: []string{},
			wantErr:      "probe cold_02.ExecTrace.Cid",
		},
		{
			// 命中库的管道报错必须上报，不能被当成「这个库里没有」
			name: "aggregate_error_is_returned",
			dbs: []dbSpec{
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_01.ExecTrace": {"Cid": true},
			},
			aggErrs: map[string]error{
				"cold_01.ExecTrace": fmt.Errorf("exceeded time limit"),
			},
			wantAggCalls: []string{"cold_01.ExecTrace"},
			wantErr:      "aggregate cold_01.ExecTrace",
		},
		{
			// 并发探测的完成顺序不能影响「假命中时先试哪个库」：候选顺序 01 → 03 必须稳定
			name: "hit_order_is_deterministic_under_concurrency",
			dbs: []dbSpec{
				db1("cold_01", smodel.Cold),
				db1("cold_02", smodel.Cold),
				db1("cold_03", smodel.Cold),
			},
			probeHits: map[string]map[string]bool{
				"cold_01.ExecTrace": {"Cid": true},
				"cold_03.ExecTrace": {"Cid": true},
			},
			// 故意让前面的库慢、后面的库快，命中顺序若不做还原就会变成 03 → 01
			probeDelay: map[string]time.Duration{
				"cold_01.ExecTrace": 30 * time.Millisecond,
				"cold_03.ExecTrace": 1 * time.Millisecond,
			},
			aggResults: map[string][]bson.M{
				"cold_03.ExecTrace": {{"Cid": "w", "IsBlock": true}},
			},
			wantAggCalls:    []string{"cold_01.ExecTrace", "cold_03.ExecTrace"},
			wantResultCount: 1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			countLists := buildCountLists(t, tc.dbs)

			probe := &fakeProbe{
				hits:  tc.probeHits,
				errs:  tc.probeErrs,
				delay: tc.probeDelay,
			}
			agg := &fakeAgg{
				results: tc.aggResults,
				errs:    tc.aggErrs,
			}

			// 用可辨识的 pipe 对象断言「原样透传、不被两阶段改写」
			pipe := []bson.M{{"$match": bson.M{"Cid": "x"}}}
			cid := "bafy2bzacedexamplecid"

			res, err := traversalQueryByCid(context.Background(), probe.probe, agg.agg, pipe, countLists, "ExecTrace", cid, CidProbeFields)

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

			if len(res) != tc.wantResultCount {
				t.Fatalf("result count = %d, want %d", len(res), tc.wantResultCount)
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

			// 管道与表名必须原样透传（响应结构由管道决定，不能被动过）
			for i := range agg.pipes {
				if agg.pipes[i] == nil || !reflect.DeepEqual(agg.pipes[i], pipe) {
					t.Fatalf("pipe #%d was modified: %v", i, agg.pipes[i])
				}
				if agg.tables[i] != "ExecTrace" {
					t.Fatalf("tableName #%d = %v, want ExecTrace", i, agg.tables[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. 边界：没有候选库 / 目标表不存在
// ---------------------------------------------------------------------------

func TestTraversalQueryByCidWithoutCandidates(t *testing.T) {
	probe := &fakeProbe{}
	agg := &fakeAgg{}

	countLists := buildCountLists(t, []dbSpec{
		{name: "cold_01", dtype: smodel.Cold, cols: []string{"Message"}},
		{name: "cold_02", dtype: smodel.Cold, cols: []string{"ExecTrace"}},
	})

	// 表不存在：没有任何候选，探测与管道都不该被调用
	res, err := traversalQueryByCid(context.Background(), probe.probe, agg.agg, []bson.M{}, countLists[:1], "ExecTrace", "cid", CidProbeFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("result = %v, want empty", res)
	}
	if calls := probe.descs(); len(calls) != 0 {
		t.Fatalf("probe must not be called, got %v", calls)
	}
	if calls := agg.called(); len(calls) != 0 {
		t.Fatalf("aggregate must not be called, got %v", calls)
	}

	// 完全空的候选列表同样安全
	res, err = traversalQueryByCid(context.Background(), probe.probe, agg.agg, []bson.M{}, nil, "ExecTrace", "cid", CidProbeFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("result = %v, want empty", res)
	}
}

// ---------------------------------------------------------------------------
// 5. 探测并发：全部候选并发发出，且结果按候选顺序还原
// ---------------------------------------------------------------------------

func TestProbeCandidatesConcurrentAndOrdered(t *testing.T) {
	countLists := buildCountLists(t, []dbSpec{
		{name: "cold_01", dtype: smodel.Cold, cols: []string{"ExecTrace"}},
		{name: "cold_02", dtype: smodel.Cold, cols: []string{"ExecTrace"}},
		{name: "cold_03", dtype: smodel.Cold, cols: []string{"ExecTrace"}},
		{name: "cold_04", dtype: smodel.Cold, cols: []string{"ExecTrace"}},
	})

	candidates := cidCandidates(countLists, "ExecTrace")

	probe := &fakeProbe{
		hits: map[string]map[string]bool{
			"cold_01.ExecTrace": {"Cid": true},
			"cold_02.ExecTrace": {"Cid": true},
			"cold_03.ExecTrace": {"Cid": true},
			"cold_04.ExecTrace": {"Cid": true},
		},
		// 越靠后的库越快完成，若不做顺序还原，hits 就会是反序
		delay: map[string]time.Duration{
			"cold_01.ExecTrace": 40 * time.Millisecond,
			"cold_02.ExecTrace": 30 * time.Millisecond,
			"cold_03.ExecTrace": 20 * time.Millisecond,
			"cold_04.ExecTrace": 10 * time.Millisecond,
		},
	}

	start := time.Now()
	hits, err := probeCandidates(context.Background(), probe.probe, candidates, "cid", CidProbeFields)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotDescs := make([]string, 0, len(hits))
	for _, hit := range hits {
		gotDescs = append(gotDescs, hit.desc)
	}

	wantDescs := []string{"cold_01.ExecTrace", "cold_02.ExecTrace", "cold_03.ExecTrace", "cold_04.ExecTrace"}
	if !reflect.DeepEqual(gotDescs, wantDescs) {
		t.Fatalf("hits = %v, want %v", gotDescs, wantDescs)
	}

	// 并发发出的证据：总耗时接近最慢的单个探测（40ms），而不是累加（100ms）
	if elapsed > 90*time.Millisecond {
		t.Fatalf("probes look serial: elapsed = %v", elapsed)
	}
}
