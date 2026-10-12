package common

import (
	"strings"
	"testing"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	rextract "github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

// TestRichListTopPipelineShape pins the query the richlist runs against the
// daily top-N collection: same predicate as the legacy ActorBalance pipeline
// (epoch equality + the three richlist codes + positive balance), same
// sort/limit shape, plus the address tiebreak that makes the ordering stable.
func TestRichListTopPipelineShape(t *testing.T) {
	const (
		epoch = int64(6447600)
		limit = int64(9223372036854775807)
	)

	v, err := util.Parse(model.Ctx{StartEpoch: epoch, EndEpoch: epoch + 1, Limit: limit}, richListTopAggregator)
	if err != nil {
		t.Fatalf("render pipeline: %v", err)
	}

	stages, ok := asArray(v)
	if !ok {
		t.Fatalf("rendered pipeline is not an array: %#v", v)
	}

	wantStages := []string{"$match", "$addFields", "$sort", "$limit", "$skip", "$limit"}
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
	if got, _ := docGet(match, "Epoch"); !numEq(got, float64(epoch)) {
		t.Fatalf("$match.Epoch = %v, want %d (ctx.StartEpoch must be injected)", got, epoch)
	}

	codes, ok := docGet(match, "Code")
	if !ok {
		t.Fatalf("$match must constrain Code, got %#v", match)
	}
	gotCodes := stringSet(t, codes)
	if len(gotCodes) != len(rextract.RichListCodes) {
		t.Fatalf("$match.Code $in = %v, want exactly %v", gotCodes, rextract.RichListCodes)
	}
	for _, want := range rextract.RichListCodes {
		if _, ok := gotCodes[want]; !ok {
			t.Fatalf("$match.Code is missing %q (must equal extract.RichListCodes = %v)", want, rextract.RichListCodes)
		}
	}

	balance, ok := docGet(match, "Balance")
	if !ok {
		t.Fatalf("$match must keep the positive-balance filter, got %#v", match)
	}
	balanceDoc := asDoc(t, balance)
	if gt, _ := docGet(balanceDoc, "$gt"); gt != "0" {
		t.Fatalf("$match.Balance.$gt = %v, want \"0\" (string compare, as upstream)", gt)
	}

	sortBody := asDoc(t, bodies[2])
	if dir, _ := docGet(sortBody, "balance"); !numEq(dir, -1) {
		t.Fatalf("$sort.balance = %v, want -1 (largest first)", dir)
	}
	if dir, _ := docGet(sortBody, "Addr"); !numEq(dir, 1) {
		t.Fatalf("$sort.Addr = %v, want 1 (stable tiebreak, matching the writer's rank)", dir)
	}

	if lmt := bodies[3]; !numEq(lmt, 1000) {
		t.Fatalf("the first $limit = %v, want 1000 (the endpoint returns 1000 rows)", lmt)
	}
	if skp := bodies[4]; !numEq(skp, 0) {
		t.Fatalf("$skip = %v, want 0", skp)
	}
}

// TestRichListTopPipelineHasNoExpr guards the one shape that would turn this
// into a collection scan on a 589GB collection.
func TestRichListTopPipelineHasNoExpr(t *testing.T) {
	v, err := util.Parse(model.Ctx{StartEpoch: 6447600, EndEpoch: 6447601}, richListTopAggregator)
	if err != nil {
		t.Fatalf("render pipeline: %v", err)
	}
	stages, ok := asArray(v)
	if !ok || len(stages) == 0 {
		t.Fatalf("not an array: %#v", v)
	}
	if hasKeyDeep(stages, "$expr") {
		t.Fatal("richlist pipeline must not use $expr")
	}
	// The epoch match is what keeps the read on an index; losing it would make
	// the read degrade to a full collection scan.
	first := asDoc(t, stages[0])[0].Value
	if _, ok := docGet(asDoc(t, first), "Epoch"); !ok {
		t.Fatal("$match must constrain Epoch")
	}
}

// TestRichListTopPipelineSwitch covers the rollback switch: local overrides on
// by default, off when LONDOBELL_AGG_PIPELINE_V2 is falsy (the upstream
// pipeline still renders against the same collection, just without the stable
// tiebreak).
func TestRichListTopPipelineSwitch(t *testing.T) {
	upstream := []byte("upstream-richlist-pipeline")

	t.Setenv(LocalPipelineV2Env, "")
	if got := RichListTopPipeline(upstream); string(got) != richListTopAggregator {
		t.Fatal("override should be enabled by default")
	}

	for _, off := range []string{"0", "false", "OFF", " no "} {
		t.Setenv(LocalPipelineV2Env, off)
		if got := RichListTopPipeline(upstream); string(got) != string(upstream) {
			t.Fatalf("env=%q should fall back to upstream, got override", off)
		}
	}
}

// TestRichListTopPipelineKeepsCtxPlaceholders guards against a pipeline that
// renders without its ctx.
func TestRichListTopPipelineKeepsCtxPlaceholders(t *testing.T) {
	const (
		epoch = int64(6447600)
		limit = int64(9223372036854775807)
	)

	v, err := util.Parse(model.Ctx{StartEpoch: epoch, EndEpoch: epoch + 1, Limit: limit}, richListTopAggregator)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	stages, ok := asArray(v)
	if !ok || len(stages) == 0 {
		t.Fatalf("not an array: %#v", v)
	}
	if !containsFloatDeep(stages, float64(epoch)) {
		t.Fatal("ctx.StartEpoch was lost during rendering")
	}
}

// stringSet reads a rendered `{$in: [...]}` array into a set.
func stringSet(t *testing.T, v interface{}) map[string]struct{} {
	t.Helper()

	doc := asDoc(t, v)
	in, ok := docGet(doc, "$in")
	if !ok {
		t.Fatalf("Code filter is not an $in list: %#v", v)
	}
	arr, ok := asArray(in)
	if !ok {
		t.Fatalf("$in is not an array: %#v", in)
	}
	out := make(map[string]struct{}, len(arr))
	for _, e := range arr {
		s, ok := e.(string)
		if !ok {
			t.Fatalf("$in element is not a string: %#v", e)
		}
		out[strings.TrimSpace(s)] = struct{}{}
	}
	return out
}
