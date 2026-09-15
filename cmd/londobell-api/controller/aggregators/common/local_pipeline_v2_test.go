package common

import (
	"encoding/json"
	"os"
	"testing"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

const (
	lookupStage = "$lookup"

	testPipelineCid = "bafy2bzaceant2nub6gteynevewkon5aeud45v4gixz3wzg32utdibuixm4zs4"
)

// renderPipeline 走生产同一条链路渲染 pipeline（otto 求值 + ctx 注入），
// 既能抓 JS 语法错误，也能验证 ctx 占位符仍然生效。
// 渲染结果是 mongodb driver 用的有序文档（bson.D / primitive.A）。
func renderPipeline(t *testing.T, pipeline []byte) primitive.A {
	t.Helper()

	v, err := util.Parse(model.Ctx{Cid: testPipelineCid, StartEpoch: 6371988}, string(pipeline))
	if err != nil {
		t.Fatalf("render pipeline: %v", err)
	}

	arr, ok := asArray(v)
	if !ok {
		t.Fatalf("rendered pipeline is not an array: %#v", v)
	}
	return arr
}

func asArray(v interface{}) (primitive.A, bool) {
	switch x := v.(type) {
	case primitive.A:
		return x, true
	case []interface{}:
		return primitive.A(x), true
	default:
		return nil, false
	}
}

func asDoc(t *testing.T, v interface{}) primitive.D {
	t.Helper()

	switch x := v.(type) {
	case primitive.D:
		return x
	case map[string]interface{}:
		d := make(primitive.D, 0, len(x))
		for k, val := range x {
			d = append(d, primitive.E{Key: k, Value: val})
		}
		return d
	default:
		t.Fatalf("not a document: %#v", v)
		return nil
	}
}

func docGet(d primitive.D, key string) (interface{}, bool) {
	for _, e := range d {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

// stageName 返回单个 pipeline 阶段的算子名（阶段必须是单键文档）。
func stageName(t *testing.T, stage interface{}) string {
	t.Helper()

	d := asDoc(t, stage)
	if len(d) != 1 {
		t.Fatalf("pipeline stage must be a single-key document, got %#v", d)
	}
	return d[0].Key
}

// collectLookups 递归收集所有 $lookup 的子文档。
func collectLookups(v interface{}, out *[]primitive.D) {
	switch x := v.(type) {
	case primitive.D:
		for _, e := range x {
			if e.Key == lookupStage {
				if d, ok := e.Value.(primitive.D); ok {
					*out = append(*out, d)
				} else if av, ok := asArray(e.Value); ok && len(av) == 1 {
					*out = append(*out, asLooseDoc(av[0]))
				}
			}
			collectLookups(e.Value, out)
		}
	case primitive.A:
		for _, e := range x {
			collectLookups(e, out)
		}
	case []interface{}:
		for _, e := range x {
			collectLookups(e, out)
		}
	case map[string]interface{}:
		for k, val := range x {
			if k == lookupStage {
				*out = append(*out, asLooseDoc(val))
			}
			collectLookups(val, out)
		}
	}
}

func asLooseDoc(v interface{}) primitive.D {
	switch x := v.(type) {
	case primitive.D:
		return x
	case map[string]interface{}:
		d := make(primitive.D, 0, len(x))
		for k, val := range x {
			d = append(d, primitive.E{Key: k, Value: val})
		}
		return d
	default:
		return nil
	}
}

// hasKeyDeep 判断文档/数组里（任意深度）是否出现某个 key。
func hasKeyDeep(v interface{}, key string) bool {
	switch x := v.(type) {
	case primitive.D:
		for _, e := range x {
			if e.Key == key {
				return true
			}
			if hasKeyDeep(e.Value, key) {
				return true
			}
		}
	case primitive.A:
		for _, e := range x {
			if hasKeyDeep(e, key) {
				return true
			}
		}
	case []interface{}:
		for _, e := range x {
			if hasKeyDeep(e, key) {
				return true
			}
		}
	case map[string]interface{}:
		for k, val := range x {
			if k == key {
				return true
			}
			if hasKeyDeep(val, key) {
				return true
			}
		}
	}
	return false
}

// TestPipelineV2Switch 覆盖开关：默认启用，置 0/false/off 回退上游。
func TestPipelineV2Switch(t *testing.T) {
	upstream := []byte("upstream-pipeline")

	t.Setenv(LocalPipelineV2Env, "")
	if got := BlocksForMessagePipeline(upstream); string(got) != blocksForMessageAggregatorV2 {
		t.Fatalf("override should be enabled by default")
	}

	for _, off := range []string{"0", "false", "OFF", " no "} {
		t.Setenv(LocalPipelineV2Env, off)
		if got := BlocksForMessagePipeline(upstream); string(got) != string(upstream) {
			t.Fatalf("env=%q should fall back to upstream, got override", off)
		}
	}

	t.Setenv(LocalPipelineV2Env, "1")
	if got := BlocksForMessagePipeline(upstream); string(got) != blocksForMessageAggregatorV2 {
		t.Fatalf("env=1 should keep the override")
	}
}

// TestBlocksForMessageOverrideIsIndexFriendly 守住 P0-A 的核心：
// 不许再出现无法走索引的 $expr join；$lookup 必须是等值 join。
func TestBlocksForMessageOverrideIsIndexFriendly(t *testing.T) {
	stages := renderPipeline(t, []byte(blocksForMessageAggregatorV2))

	want := []string{"$match", "$group", "$lookup", "$unwind", "$project"}
	if len(stages) != len(want) {
		t.Fatalf("stage count = %d, want %d", len(stages), len(want))
	}
	for i, name := range want {
		if got := stageName(t, stages[i]); got != name {
			t.Fatalf("stage[%d] = %s, want %s", i, got, name)
		}
	}

	var lookups []primitive.D
	collectLookups(stages, &lookups)
	if len(lookups) != 1 {
		t.Fatalf("want exactly 1 $lookup, got %d", len(lookups))
	}

	lk := lookups[0]
	if from, _ := docGet(lk, "from"); from != "BlockHeader" {
		t.Fatalf("$lookup.from = %v, want BlockHeader", from)
	}
	if local, _ := docGet(lk, "localField"); local != "Blocks" {
		t.Fatalf("$lookup.localField = %v, want Blocks", local)
	}
	if foreign, _ := docGet(lk, "foreignField"); foreign != "_id" {
		t.Fatalf("$lookup.foreignField = %v, want _id", foreign)
	}
	if _, ok := docGet(lk, "pipeline"); ok {
		t.Fatalf("$lookup must not carry a sub pipeline: %#v", lk)
	}
	if _, ok := docGet(lk, "let"); ok {
		t.Fatalf("$lookup must not use let/$$ vars: %#v", lk)
	}

	if hasKeyDeep(stages, "$expr") {
		t.Fatalf("override still contains $expr, which cannot use the _id_ index")
	}
}

// TestBlocksForMessageKeepsOutputShape 比算子序列与输出投影与上游一致（结果条数与字段都不许变）。
func TestBlocksForMessageKeepsOutputShape(t *testing.T) {
	upstreamStages := renderPipeline(t, monitor.GetBlocksForMessageAggregator())
	overrideStages := renderPipeline(t, []byte(blocksForMessageAggregatorV2))

	if len(upstreamStages) != len(overrideStages) {
		t.Fatalf("stage count changed: upstream=%d override=%d", len(upstreamStages), len(overrideStages))
	}
	for i := range upstreamStages {
		if stageName(t, upstreamStages[i]) != stageName(t, overrideStages[i]) {
			t.Fatalf("stage[%d] name changed", i)
		}
	}

	for i := range upstreamStages {
		name := stageName(t, upstreamStages[i])
		if name == lookupStage {
			continue // 这一条就是被替换掉的算子，由上面那个用例负责断言
		}
		upBody := stageBodyRaw(t, upstreamStages[i])
		opBody := stageBodyRaw(t, overrideStages[i])
		if !deepEqualLoose(upBody, opBody) {
			t.Fatalf("stage[%d] (%s) changed: upstream=%#v override=%#v", i, name, upBody, opBody)
		}
	}
}

// stageBodyRaw 返回阶段的原始 body（可能是有序文档，也可能是 "$field" 这类标量）。
func stageBodyRaw(t *testing.T, stage interface{}) interface{} {
	t.Helper()

	d := asDoc(t, stage)
	if len(d) != 1 {
		t.Fatalf("pipeline stage must be a single-key document, got %#v", d)
	}
	if doc, ok := d[0].Value.(primitive.D); ok {
		return doc
	}
	return d[0].Value
}

// TestChildTransfersStaysUpstream 记录 2026-09-15 的生产实测结论，防止有人再去“优化”它：
// 上游 child_transfers 的 $lookup（$indexOfBytes 前缀匹配）里同时带 {$eq:["$Epoch","$$epoch"]}，
// MongoDB 4.4 上已能用 Epoch_1 索引先定位同 epoch 的少量文档（实测 6 个真实 parent：4~12ms），
// 前缀判断只在这批文档上做内存过滤；改成 Epoch 等值 join 反而要多搬整个 epoch（实测 1061 条）的文档。
func TestChildTransfersStaysUpstream(t *testing.T) {
	stages := renderPipeline(t, monitor.GetChildTransfersForMessageAggregator())

	var lookups []primitive.D
	collectLookups(stages, &lookups)

	var prefixLookup *primitive.D
	for i := range lookups {
		if hasKeyDeep(lookups[i], "$indexOfBytes") {
			prefixLookup = &lookups[i]
			break
		}
	}
	if prefixLookup == nil {
		t.Fatalf("上游 pipeline 结构变了：找不到带 $indexOfBytes 的 $lookup；" +
			"若确有改动请先用真实数据在生产库上做 explain/计时对比再更新本用例")
	}
	if !hasKeyDeep(*prefixLookup, "$eq") {
		t.Fatalf("上游 pipeline 结构变了：前缀 $lookup 里不再有等值谓词（这是它能走索引的原因）")
	}
}

// TestOverridesKeepCtxPlaceholders 确保覆盖版仍然引用 ctx 占位符（否则会退化成全库扫描）。
func TestOverridesKeepCtxPlaceholders(t *testing.T) {
	v, err := util.Parse(model.Ctx{Cid: testPipelineCid, StartEpoch: 6371988}, string([]byte(blocksForMessageAggregatorV2)))
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	stages, ok := asArray(v)
	if !ok || len(stages) == 0 {
		t.Fatalf("not an array")
	}
	if !containsStringDeep(stages, testPipelineCid) {
		t.Fatalf("ctx.Cid was lost during rendering")
	}
	if !containsFloatDeep(stages, 6371988) {
		t.Fatalf("ctx.StartEpoch was lost during rendering")
	}
}

func containsStringDeep(v interface{}, want string) bool {
	switch x := v.(type) {
	case string:
		return x == want
	case primitive.D:
		for _, e := range x {
			if containsStringDeep(e.Value, want) {
				return true
			}
		}
	case primitive.A:
		for _, e := range x {
			if containsStringDeep(e, want) {
				return true
			}
		}
	case []interface{}:
		for _, e := range x {
			if containsStringDeep(e, want) {
				return true
			}
		}
	}
	return false
}

func containsFloatDeep(v interface{}, want float64) bool {
	switch x := v.(type) {
	case float64:
		return x == want
	case int64:
		return float64(x) == want
	case int:
		return float64(x) == want
	case primitive.D:
		for _, e := range x {
			if containsFloatDeep(e.Value, want) {
				return true
			}
		}
	case primitive.A:
		for _, e := range x {
			if containsFloatDeep(e, want) {
				return true
			}
		}
	case []interface{}:
		for _, e := range x {
			if containsFloatDeep(e, want) {
				return true
			}
		}
	}
	return false
}

// deepEqualLoose 比较渲染结果（可能是 primitive.D/A 或标量）。
func deepEqualLoose(a, b interface{}) bool {
	if ad, ok := a.(primitive.D); ok {
		bd, ok := b.(primitive.D)
		if !ok || len(ad) != len(bd) {
			return false
		}
		for i := range ad {
			if ad[i].Key != bd[i].Key || !deepEqualLoose(ad[i].Value, bd[i].Value) {
				return false
			}
		}
		return true
	}
	if aa, ok := asArray(a); ok {
		ba, ok := asArray(b)
		if !ok || len(aa) != len(ba) {
			return false
		}
		for i := range aa {
			if !deepEqualLoose(aa[i], ba[i]) {
				return false
			}
		}
		return true
	}
	return a == b
}

// toPlain 把渲染结果（bson.D/primitive.A）转成普通 JSON 结构，便于拿去 mongo 端做 explain 核对。
func toPlain(v interface{}) interface{} {
	switch x := v.(type) {
	case primitive.D:
		m := make(map[string]interface{}, len(x))
		for _, e := range x {
			m[e.Key] = toPlain(e.Value)
		}
		return m
	case primitive.A:
		out := make([]interface{}, 0, len(x))
		for _, e := range x {
			out = append(out, toPlain(e))
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(x))
		for _, e := range x {
			out = append(out, toPlain(e))
		}
		return out
	default:
		return v
	}
}

// TestDumpRenderedPipelines 仅用于运维核对（默认 skip）：
//
//	DUMP_PIPELINES=1 go test -run TestDumpRenderedPipelines -v ./cmd/londobell-api/controller/aggregators/common/
//
// 打印覆盖版与上游版的渲染结果，用于在生产 mongo 上跑 explain / 计时验证（本仓改动即用此法验证过）。
func TestDumpRenderedPipelines(t *testing.T) {
	if os.Getenv("DUMP_PIPELINES") == "" {
		t.Skip("set DUMP_PIPELINES=1 to dump rendered pipelines")
	}

	cases := []struct {
		name     string
		pipeline []byte
	}{
		{"blocks_for_message.override", []byte(blocksForMessageAggregatorV2)},
		{"blocks_for_message.upstream", monitor.GetBlocksForMessageAggregator()},
		{"child_transfers_for_message.upstream", monitor.GetChildTransfersForMessageAggregator()},
	}

	for _, tc := range cases {
		v, err := util.Parse(model.Ctx{Cid: testPipelineCid, StartEpoch: 6371988}, string(tc.pipeline))
		if err != nil {
			t.Fatalf("%s: render: %v", tc.name, err)
		}
		raw, err := json.Marshal(toPlain(v))
		if err != nil {
			t.Fatalf("%s: marshal: %v", tc.name, err)
		}
		t.Logf("PIPELINE %s %s", tc.name, raw)
	}
}
