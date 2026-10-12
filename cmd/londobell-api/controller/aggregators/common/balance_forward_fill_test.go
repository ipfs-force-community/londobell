package common

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// TestBalanceForwardFillPipelineShape pins the query shape the forward-fill
// depends on: Epoch in [StartEpoch, EndEpoch) + Addr, newest Epoch first,
// exactly one row. A point lookup (the upstream shape) would miss every actor
// that did not move on a delta tick.
func TestBalanceForwardFillPipelineShape(t *testing.T) {
	const (
		start = int64(6369840)
		end   = int64(6372001)
		addr  = "f01234"
	)

	v, err := util.Parse(model.Ctx{Addr: addr, StartEpoch: start, EndEpoch: end}, balanceForwardFillAggregator)
	if err != nil {
		t.Fatalf("render pipeline: %v", err)
	}

	stages, ok := asArray(v)
	if !ok {
		t.Fatalf("rendered pipeline is not an array: %#v", v)
	}

	wantStages := []string{"$match", "$sort", "$limit"}
	if len(stages) != len(wantStages) {
		t.Fatalf("stage count = %d, want %d", len(stages), len(wantStages))
	}
	bodies := make([]interface{}, 0, len(stages))
	for i, name := range wantStages {
		if got := stageName(t, stages[i]); got != name {
			t.Fatalf("stage[%d] = %s, want %s", i, got, name)
		}
		bodies = append(bodies, asDoc(t, stages[i])[0].Value)
	}

	match := asDoc(t, bodies[0])
	if got, _ := docGet(match, "Addr"); got != addr {
		t.Fatalf("$match.Addr = %v, want %q (ctx.Addr must be injected)", got, addr)
	}

	epochRange, ok := docGet(match, "Epoch")
	if !ok {
		t.Fatalf("$match must constrain Epoch, got %#v", match)
	}
	epochDoc := asDoc(t, epochRange)
	if gte, _ := docGet(epochDoc, "$gte"); !numEq(gte, float64(start)) {
		t.Fatalf("$match.Epoch.$gte = %v, want %d", gte, start)
	}
	if lt, _ := docGet(epochDoc, "$lt"); !numEq(lt, float64(end)) {
		t.Fatalf("$match.Epoch.$lt = %v, want %d", lt, end)
	}
	if _, ok := docGet(epochDoc, "$lte"); ok {
		t.Fatalf("$match.Epoch must be a half-open [start,end) range: %#v", epochDoc)
	}
	if _, ok := docGet(epochDoc, "$eq"); ok {
		t.Fatalf("$match.Epoch must not be an exact-epoch point lookup: %#v", epochDoc)
	}

	sortBody := asDoc(t, bodies[1])
	if dir, _ := docGet(sortBody, "Epoch"); !numEq(dir, -1) {
		t.Fatalf("$sort.Epoch = %v, want -1 (newest first)", dir)
	}

	if lmt := bodies[2]; !numEq(lmt, 1) {
		t.Fatalf("$limit = %v, want 1", lmt)
	}

	if hasKeyDeep(stages, "$expr") {
		t.Fatalf("forward-fill pipeline must not use $expr")
	}
}

// TestBalanceForwardFillPipelineSwitch covers the rollback switch: local
// overrides on by default, off when LONDOBELL_AGG_PIPELINE_V2 is falsy.
func TestBalanceForwardFillPipelineSwitch(t *testing.T) {
	upstream := []byte("upstream-balance-pipeline")

	t.Setenv(LocalPipelineV2Env, "")
	if got := BalanceForwardFillPipeline(upstream); string(got) != balanceForwardFillAggregator {
		t.Fatalf("override should be enabled by default")
	}

	for _, off := range []string{"0", "false", "OFF", " no "} {
		t.Setenv(LocalPipelineV2Env, off)
		if got := BalanceForwardFillPipeline(upstream); string(got) != string(upstream) {
			t.Fatalf("env=%q should fall back to upstream, got override", off)
		}
	}
}

// TestBalanceForwardFillKeepsCtxPlaceholders guards against a pipeline that
// renders without its ctx (which would degrade into a full-collection scan).
func TestBalanceForwardFillKeepsCtxPlaceholders(t *testing.T) {
	const (
		start = int64(6369840)
		end   = int64(6372001)
		addr  = "f01234"
	)

	v, err := util.Parse(model.Ctx{Addr: addr, StartEpoch: start, EndEpoch: end}, balanceForwardFillAggregator)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	stages, ok := asArray(v)
	if !ok || len(stages) == 0 {
		t.Fatalf("not an array: %#v", v)
	}

	if !containsStringDeep(stages, addr) {
		t.Fatalf("ctx.Addr was lost during rendering")
	}
	if !containsFloatDeep(stages, float64(start)) {
		t.Fatalf("ctx.StartEpoch was lost during rendering")
	}
	if !containsFloatDeep(stages, float64(end)) {
		t.Fatalf("ctx.EndEpoch was lost during rendering")
	}
}

// numEq compares a rendered numeric against a wanted float.
func numEq(v interface{}, want float64) bool {
	switch x := v.(type) {
	case float64:
		return x == want
	case int64:
		return float64(x) == want
	case int:
		return float64(x) == want
	case int32:
		return float64(x) == want
	case primitive.Decimal128:
		return false
	default:
		return false
	}
}
