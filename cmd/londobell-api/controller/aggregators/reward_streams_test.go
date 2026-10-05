package aggregators

import (
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// 以下两个 fixture 是 2026-10-05 经跳板机对生产 londobell 的只读抓取原样片段
// （POST /aggregators/actor_state_epoch），未做任何编造或数字改写。
//
// Cali（NV29 已上线）：fil/19/reward，含 TotalMintedReward/TotalBurnMinted/TotalExplicitMinted。
const caliV19Fixture = `{"code":0,"msg":"","data":[{"Addr":"02","Code":"fil/19/reward","Balance":"989045298642071010695037639","Epoch":4129200,"Detail":{"Accrued":[{"Amount":"43474537298093792975065","ID":2}],"CumsumBaseline":"4422231777257704874428","CumsumRealized":"4420434271779304767488","EffectiveBaselinePower":"2891804839359069189","EffectiveNetworkTime":1530,"Epoch":4129200,"SWAActor":"0200118","SWATimelockEpochs":720,"StreamsRoot":"bafy2bzacedx5alynikpa32weqwxdzwco35jftdpobcx4yhw7ihxkl2niffvqh4","ThisEpochBaselinePower":"43974520843908190170","ThisEpochReward":"23040518119342820469","ThisEpochRewardSmoothed":{"PositionEstimate":"7840281893995891653076480719666523554613657611336443785992","VelocityEstimate":"-861715248219431356114421686686105715278631607078190"},"TotalBurnMinted":"24043163217827762771763","TotalExplicitMinted":"77211820386747518854736","TotalMintedReward":"110958573454974240267760798"}}]}`

// 主网（NV29 未排期）：fil/18/reward，只有 TotalStoragePowerReward。
const mainnetV18Fixture = `{"code":0,"msg":"","data":[{"Addr":"02","Code":"fil/18/reward","Balance":"690663841996126139104197526","Epoch":6429840,"Detail":{"BaselineTotal":"768335872210768889362796814","CumsumBaseline":"43798288958982039424395743","CumsumRealized":"43798278896301797563559570","EffectiveBaselinePower":"31768883652690369911","EffectiveNetworkTime":3636130,"Epoch":6429840,"SimpleTotal":"330000000000000000000000000","ThisEpochBaselinePower":"200461368555073534757","ThisEpochReward":"20571896745030370974","ThisEpochRewardSmoothed":{"PositionEstimate":"7000708744241635192262388634371023560054042267334923271979","VelocityEstimate":"-1434862497058962418763251096020120012196463962522160"},"TotalStoragePowerReward":"409336223003873867555802474"}}]}`

// decodeFixture 走与 handler 同一条 json 解码路径（Epoch + Detail map，多余字段忽略）。
func decodeFixture(t *testing.T, raw string) []rewardStreamRow {
	t.Helper()

	var envelope struct {
		Code uint64            `json:"code"`
		Data []rewardStreamRow `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Fatalf("fixture decoded to 0 rows")
	}
	return envelope.Data
}

// (b) v19 行 → 四字段齐全，且大平滑估计串没有混进来（Detail 只解出四个计数器里存在的）。
func TestBuildRewardStreamsV19RealSample(t *testing.T) {
	rows := decodeFixture(t, caliV19Fixture)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}

	got := buildRewardStreams(rows)
	if len(got) != 1 {
		t.Fatalf("want 1 point, got %d", len(got))
	}

	want := model.RewardStreamPoint{
		Epoch:                   4129200,
		TotalStoragePowerReward: "0", // v19 无此计数器
		TotalMintedReward:       "110958573454974240267760798",
		TotalBurnMinted:         "24043163217827762771763",
		TotalExplicitMinted:     "77211820386747518854736",
	}
	if got[0] != want {
		t.Fatalf("v19 point mismatch:\n got=%+v\nwant=%+v", got[0], want)
	}
}

// (a) v18 行 → 三项新字段为 "0"，TotalStoragePowerReward 保留原值。
func TestBuildRewardStreamsV18RealSample(t *testing.T) {
	rows := decodeFixture(t, mainnetV18Fixture)
	got := buildRewardStreams(rows)
	if len(got) != 1 {
		t.Fatalf("want 1 point, got %d", len(got))
	}

	want := model.RewardStreamPoint{
		Epoch:                   6429840,
		TotalStoragePowerReward: "409336223003873867555802474",
		TotalMintedReward:       "0",
		TotalBurnMinted:         "0",
		TotalExplicitMinted:     "0",
	}
	if got[0] != want {
		t.Fatalf("v18 point mismatch:\n got=%+v\nwant=%+v", got[0], want)
	}
}

// (c) 空结果 → 空序列（长度为 0，不是 nil），不做前值填充。
func TestBuildRewardStreamsEmpty(t *testing.T) {
	got := buildRewardStreams(nil)
	if got == nil {
		t.Fatalf("empty result must be a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("want 0 points, got %d", len(got))
	}
}

// 乱序输入 → 输出按 Epoch 升序（多库合并后顺序不保证）。
func TestBuildRewardStreamsSortsAscending(t *testing.T) {
	got := buildRewardStreams([]rewardStreamRow{
		{Epoch: 4129320, Detail: map[string]interface{}{"TotalMintedReward": "3"}},
		{Epoch: 4129200, Detail: map[string]interface{}{"TotalMintedReward": "1"}},
		{Epoch: 4129080, Detail: map[string]interface{}{"TotalMintedReward": "0"}},
	})

	wantEpochs := []int64{4129080, 4129200, 4129320}
	for i, w := range wantEpochs {
		if got[i].Epoch != w {
			t.Fatalf("points[%d].Epoch = %d, want %d (order must be ascending)", i, got[i].Epoch, w)
		}
	}
}

// 缺字段/空串/非字符串一律归一成 "0"，四字段恒定存在。
func TestCounterValueDefaults(t *testing.T) {
	if v := counterValue(nil, "TotalBurnMinted"); v != "0" {
		t.Fatalf("missing detail -> %q, want \"0\"", v)
	}
	if v := counterValue(map[string]interface{}{"TotalBurnMinted": ""}, "TotalBurnMinted"); v != "0" {
		t.Fatalf("empty string -> %q, want \"0\"", v)
	}
	if v := counterValue(map[string]interface{}{"TotalBurnMinted": nil}, "TotalBurnMinted"); v != "0" {
		t.Fatalf("nil value -> %q, want \"0\"", v)
	}
	if v := counterValue(map[string]interface{}{"TotalBurnMinted": int64(7)}, "TotalBurnMinted"); v != "7" {
		t.Fatalf("non-string -> %q, want \"7\"", v)
	}
	if v := counterValue(map[string]interface{}{"TotalBurnMinted": "42"}, "TotalBurnMinted"); v != "42" {
		t.Fatalf("present value -> %q, want \"42\"", v)
	}
}

// (d) 超上限报错：2000 行放行，2001 行（管道多取的那一行）必须报错。
func TestCheckRewardStreamsRowsLimit(t *testing.T) {
	if err := checkRewardStreamsRows(maxRewardStreamsRows); err != nil {
		t.Fatalf("%d rows must pass, got error: %v", maxRewardStreamsRows, err)
	}
	if err := checkRewardStreamsRows(0); err != nil {
		t.Fatalf("0 rows must pass, got error: %v", err)
	}
	err := checkRewardStreamsRows(maxRewardStreamsRows + 1)
	if err == nil {
		t.Fatalf("%d rows must fail the hard limit", maxRewardStreamsRows+1)
	}
	if got := err.Error(); got == "" {
		t.Fatalf("limit error must carry an explicit message")
	}
}

// 管道形状：$match(Epoch 区间 + Addr) → $project(点号投影四个计数器) → $sort → $limit。
// 同时钉住「不吐整个 Detail、不吐平滑估计串」与行数上限。
func TestRewardStreamsPipelineShape(t *testing.T) {
	const (
		start = 4129200
		end   = 4152000
	)

	v, err := util.Parse(model.Ctx{Addr: rewardActorAddr, StartEpoch: start, EndEpoch: end}, string(rewardStreamsAggregator))
	if err != nil {
		t.Fatalf("render pipeline: %v", err)
	}

	stages, ok := testAsArray(v)
	if !ok {
		t.Fatalf("rendered pipeline is not an array: %#v", v)
	}

	wantStages := []string{"$match", "$project", "$sort", "$limit"}
	if len(stages) != len(wantStages) {
		t.Fatalf("stage count = %d, want %d", len(stages), len(wantStages))
	}
	stageBodies := make([]interface{}, 0, len(stages))
	for i, name := range wantStages {
		got, body := testStageKV(t, stages[i])
		if got != name {
			t.Fatalf("stage[%d] = %s, want %s", i, got, name)
		}
		stageBodies = append(stageBodies, body)
	}

	match := testDoc(t, stageBodies[0])

	// Addr 必须渲染出裸串 "02"（ctx.Addr 注入生效）。
	if addr, _ := testDocGet(match, "Addr"); addr != rewardActorAddr {
		t.Fatalf("$match.Addr = %v, want %q", addr, rewardActorAddr)
	}

	epochRange, ok := testDocGet(match, "Epoch")
	if !ok {
		t.Fatalf("$match must constrain Epoch (index prefix), got %#v", match)
	}
	epochDoc := testDoc(t, epochRange)
	if gte, _ := testDocGet(epochDoc, "$gte"); !testNumEq(gte, start) {
		t.Fatalf("$match.Epoch.$gte = %v, want %d", gte, start)
	}
	if lt, _ := testDocGet(epochDoc, "$lt"); !testNumEq(lt, end) {
		t.Fatalf("$match.Epoch.$lt = %v, want %d", lt, end)
	}
	if _, ok := testDocGet(epochDoc, "$gt"); ok {
		t.Fatalf("$match.Epoch must be [start,end) (no $gt): %#v", epochDoc)
	}

	project := testDoc(t, stageBodies[1])
	wantProjectKeys := []string{
		"_id",
		"Epoch",
		"Detail.TotalStoragePowerReward",
		"Detail.TotalMintedReward",
		"Detail.TotalBurnMinted",
		"Detail.TotalExplicitMinted",
	}
	if len(project) != len(wantProjectKeys) {
		t.Fatalf("$project key set changed: got %d keys %#v, want %d", len(project), project, len(wantProjectKeys))
	}
	for _, k := range wantProjectKeys {
		if _, ok := testDocGet(project, k); !ok {
			t.Fatalf("$project missing key %q: %#v", k, project)
		}
	}
	// 不许吐整个 Detail / 大平滑估计串。
	for _, forbidden := range []string{"Detail", "ThisEpochRewardSmoothed"} {
		if _, ok := testDocGet(project, forbidden); ok {
			t.Fatalf("$project must not include %q: %#v", forbidden, project)
		}
	}

	sortBody := testDoc(t, stageBodies[2])
	if dir, _ := testDocGet(sortBody, "Epoch"); !testNumEq(dir, 1) {
		t.Fatalf("$sort.Epoch = %v, want 1 (ascending)", dir)
	}

	lmt := stageBodies[3] // $limit 的 body 是标量（行数）
	if !testNumEq(lmt, rewardStreamsPipelineLimit) {
		t.Fatalf("$limit = %v, want %d (hard limit + 1)", lmt, rewardStreamsPipelineLimit)
	}
	if rewardStreamsPipelineLimit != maxRewardStreamsRows+1 {
		t.Fatalf("pipeline limit must be hard limit + 1")
	}
}

// ---- 渲染结果断言用的小工具（bson.D / primitive.A）----

// testAsArray 兼容 otto 渲染结果的两种数组形态。
func testAsArray(v interface{}) ([]interface{}, bool) {
	switch x := v.(type) {
	case primitive.A:
		return []interface{}(x), true
	case []interface{}:
		return x, true
	default:
		return nil, false
	}
}

// testStageKV 要求单个 pipeline 阶段是单键文档，返回该键与其 body。
func testStageKV(t *testing.T, stage interface{}) (string, interface{}) {
	t.Helper()

	d := testDoc(t, stage)
	if len(d) != 1 {
		t.Fatalf("pipeline stage must be a single-key document, got %#v", d)
	}
	return d[0].Key, d[0].Value
}

// testDocGet 在有序文档里按 key 取值。
func testDocGet(d primitive.D, key string) (interface{}, bool) {
	for _, e := range d {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

func testDoc(t *testing.T, v interface{}) primitive.D {
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

func testNumEq(v interface{}, want float64) bool {
	switch x := v.(type) {
	case float64:
		return x == want
	case int64:
		return float64(x) == want
	case int:
		return float64(x) == want
	case int32:
		return float64(x) == want
	default:
		return false
	}
}
