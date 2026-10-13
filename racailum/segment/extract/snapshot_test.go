package extract

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"github.com/ipfs-force-community/londobell/common"
)

// mainnet / production cadence: interval = 1h, ActorBalanceTicks = 24 -> 1 day.
const (
	testInterval = builtin.EpochsInHour // 120
	testTicks    = 24
	testTick     = abi.ChainEpoch(testTicks) * testInterval // 2880
)

// zeroHourResidue is the smallest non-negative epoch that is a local midnight.
// Under the mainnet build this is 2160 (genesis 2020-08-25T06:00:00+08:00), and
// it repeats every epochsPerDay. Derived here instead of hardcoded so the
// suite stays honest under the calibnet build tag / a different genesis time.
func zeroHourResidue(t *testing.T) abi.ChainEpoch {
	t.Helper()

	for h := abi.ChainEpoch(0); h < epochsPerDay; h++ {
		if common.IsZeroHour(true, h) {
			return h
		}
	}

	t.Fatalf("no zero hour found in the first %d epochs", epochsPerDay)
	return 0
}

// TestSnapshotCadence pins the arithmetic the whole optimisation leans on:
// ticks every 24h, zero hours every 24h but offset by the genesis hour, so a
// given epoch is never both.
func TestSnapshotCadence(t *testing.T) {
	zh := zeroHourResidue(t)
	t.Logf("zero hour residue = %d (mod %d)", zh, epochsPerDay)

	if TickInterval(testTicks, testInterval) != testTick {
		t.Fatalf("tick interval = %d, want %d", TickInterval(testTicks, testInterval), testTick)
	}
	if epochsPerDay != testTick {
		t.Fatalf("a local day must be %d epochs, got %d", testTick, epochsPerDay)
	}
	if IsTickEpoch(testTicks, testInterval, zh) {
		t.Fatalf("zero hour %d must not also be a tick epoch (modes must not overlap)", zh)
	}
	if common.IsZeroHour(true, testTick) {
		t.Fatalf("tick epoch %d must not also be a zero hour (modes must not overlap)", testTick)
	}
}

// TestIsTickEpochMatchesIsExtract keeps the ctx-free twin honest against the
// real gate helper: if they ever diverge the writer would run on a different
// schedule than the rest of the extractor.
func TestIsTickEpochMatchesIsExtract(t *testing.T) {
	ctx := &Ctx{Opts: Options{StateRegular: ActorStateRegularOptions{Interval: testInterval}}}

	for _, ticks := range []int{testTicks, 1, 0, -1} {
		for h := abi.ChainEpoch(0); h < 3*testTick; h++ {
			want := IsExtract(ticks, ctx, h)
			got := IsTickEpoch(ticks, testInterval, h)
			if got != want {
				t.Fatalf("ticks=%d height=%d: IsTickEpoch=%v, IsExtract=%v", ticks, h, got, want)
			}
		}
	}
}

// TestPreviousSnapshotHeight covers the zero-hour/tick interleaving: the tick
// right after a zero hour must diff against that zero hour, not against the
// previous tick a whole day back.
func TestPreviousSnapshotHeight(t *testing.T) {
	zh := zeroHourResidue(t)

	cases := []struct {
		name     string
		height   abi.ChainEpoch
		ticks    int
		interval abi.ChainEpoch
		zeroHour bool
		want     abi.ChainEpoch
		wantOK   bool
	}{
		{name: "first tick of day 1 -> the zero hour 720 epochs before", height: testTick, ticks: testTicks, interval: testInterval, zeroHour: true, want: zh, wantOK: true},
		{name: "tick of day 2 -> zero hour of day 2", height: 2 * testTick, ticks: testTicks, interval: testInterval, zeroHour: true, want: zh + testTick, wantOK: true},
		{name: "tick of day 3 -> zero hour of day 3", height: 3 * testTick, ticks: testTicks, interval: testInterval, zeroHour: true, want: zh + 2*testTick, wantOK: true},
		{name: "odd height between tick and zero hour -> the tick below it", height: 2*testTick + 100, ticks: testTicks, interval: testInterval, zeroHour: true, want: 2 * testTick, wantOK: true},
		{name: "odd height after a zero hour -> that zero hour", height: zh + testTick + 100, ticks: testTicks, interval: testInterval, zeroHour: true, want: zh + testTick, wantOK: true},
		{name: "without zero hours the previous tick is the base", height: 2 * testTick, ticks: testTicks, interval: testInterval, zeroHour: false, want: testTick, wantOK: true},
		{name: "no previous snapshot at genesis", height: 0, ticks: testTicks, interval: testInterval, zeroHour: true, want: 0, wantOK: false},
		{name: "no previous snapshot below genesis", height: -7, ticks: testTicks, interval: testInterval, zeroHour: true, want: 0, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := PreviousSnapshotHeight(tc.height, tc.ticks, tc.interval, tc.zeroHour)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Fatalf("prev = %d, want %d", got, tc.want)
			}
			if got >= tc.height && ok {
				t.Fatalf("previous snapshot %d must be strictly below height %d", got, tc.height)
			}
		})
	}
}

// TestLatestTopSnapshotHeight covers the richlist endpoint's epoch lookup:
// it must land on a local midnight (a daily top-N row-set).
func TestLatestTopSnapshotHeight(t *testing.T) {
	zh := zeroHourResidue(t)

	cases := []struct {
		name     string
		height   abi.ChainEpoch
		zeroHour bool
		want     abi.ChainEpoch
		wantOK   bool
	}{
		{name: "on the zero hour itself", height: zh, zeroHour: true, want: zh, wantOK: true},
		{name: "just after a zero hour -> that zero hour", height: zh + 1, zeroHour: true, want: zh, wantOK: true},
		{name: "on a tick -> the zero hour of the same day", height: testTick, zeroHour: true, want: zh, wantOK: true},
		{name: "next day's zero hour", height: testTick + zh, zeroHour: true, want: testTick + zh, wantOK: true},
		{name: "zero-hour extraction off -> no top-N snapshot", height: 2*testTick + 900, zeroHour: false, want: 0, wantOK: false},
		{name: "negative height", height: -1, zeroHour: true, want: 0, wantOK: false},
		{name: "before the first zero hour", height: zh - 1, zeroHour: true, want: 0, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LatestTopSnapshotHeight(tc.height, tc.zeroHour)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
			if ok && got > tc.height {
				t.Fatalf("top-N snapshot %d must not be above the requested height %d", got, tc.height)
			}
			if ok && !IsTopSnapshotEpoch(tc.zeroHour, got) {
				t.Fatalf("top-N snapshot %d is not a top snapshot epoch", got)
			}
		})
	}
}

// TestIsTopSnapshotEpoch pins the daily cadence and the off switch: the top-N
// snapshot is written at the local midnight and nowhere else, and never when
// zero-hour extraction is disabled.
func TestIsTopSnapshotEpoch(t *testing.T) {
	zh := zeroHourResidue(t)

	for h := abi.ChainEpoch(0); h < 3*testTick; h++ {
		want := h%epochsPerDay == zh
		if got := IsTopSnapshotEpoch(true, h); got != want {
			t.Fatalf("height=%d: IsTopSnapshotEpoch(true)=%v, want %v", h, got, want)
		}
		if IsTopSnapshotEpoch(false, h) {
			t.Fatalf("height=%d: top-N must be off when zero-hour extraction is off", h)
		}
	}
}

// TestEffectiveActorBalanceTopN pins the unset-config behaviour: an existing
// config file predates the field and decodes to 0, which must mean the default
// rather than "write nothing".
func TestEffectiveActorBalanceTopN(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{name: "unset", in: 0, want: DefaultActorBalanceTopN},
		{name: "negative", in: -5, want: DefaultActorBalanceTopN},
		{name: "configured", in: 250, want: 250},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectiveActorBalanceTopN(tc.in); got != tc.want {
				t.Fatalf("EffectiveActorBalanceTopN(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}

	// DefaultOptions must carry the default so a freshly written config still
	// produces a top-N snapshot.
	if got := DefaultOptions().ActorBalanceTopN; got != DefaultActorBalanceTopN {
		t.Fatalf("DefaultOptions().ActorBalanceTopN = %d, want %d", got, DefaultActorBalanceTopN)
	}
}

// TestPlanActorBalanceWrite is the write-side decision table: the zero hour and
// a regular tick both write only the changed actors, everything else skips, and
// any case without a usable base -> full (a re-anchor, not the daily cadence).
func TestPlanActorBalanceWrite(t *testing.T) {
	zh := zeroHourResidue(t)

	cases := []struct {
		name     string
		height   abi.ChainEpoch
		ticks    int
		interval abi.ChainEpoch
		zeroHour bool
		enabled  bool
		wantMode ActorBalanceMode
		wantPrev abi.ChainEpoch
	}{
		{name: "zero hour -> changed against the previous snapshot, not full", height: zh, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceChanged, wantPrev: 0},
		{name: "zero hour next day -> changed against the previous tick", height: zh + testTick, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceChanged, wantPrev: testTick},
		{name: "regular tick -> changed against the zero hour", height: testTick, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceChanged, wantPrev: zh},
		{name: "regular tick day 2 -> changed", height: 2 * testTick, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceChanged, wantPrev: zh + testTick},
		{name: "neither zero hour nor tick -> skip", height: 1000, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceSkip},
		{name: "tick + 1 -> skip", height: testTick + 1, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceSkip},
		{name: "extraction disabled -> skip", height: zh, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: false, wantMode: ActorBalanceSkip},
		{name: "first snapshot (no previous) -> full re-anchor", height: 0, ticks: testTicks, interval: testInterval, zeroHour: true, enabled: true, wantMode: ActorBalanceFull},
		{name: "no zero-hour base at all -> full (legacy fallback)", height: testTick, ticks: testTicks, interval: testInterval, zeroHour: false, enabled: true, wantMode: ActorBalanceFull},
		{name: "stale base one tick interval back -> full", height: 7000, ticks: 10, interval: 100, zeroHour: true, enabled: true, wantMode: ActorBalanceFull},
		{name: "recent zero-hour base -> changed", height: 6000, ticks: 10, interval: 100, zeroHour: true, enabled: true, wantMode: ActorBalanceChanged, wantPrev: 5040},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := PlanActorBalanceWrite(tc.height, tc.ticks, tc.interval, tc.zeroHour, tc.enabled)
			if plan.Mode != tc.wantMode {
				t.Fatalf("mode = %s, want %s", plan.Mode, tc.wantMode)
			}
			if tc.wantMode == ActorBalanceChanged && plan.PrevHeight != tc.wantPrev {
				t.Fatalf("prev = %d, want %d", plan.PrevHeight, tc.wantPrev)
			}
		})
	}
}

// TestZeroHourWritesChangesOnly pins requirement 1: the midnight snapshot must
// not write a full row-set any more. In the production cadence (zero-hour
// extraction on, one tick per day) every snapshot epoch inside three days must
// be a delta — the only full writes left are the no-base re-anchor.
func TestZeroHourWritesChangesOnly(t *testing.T) {
	sawZeroHour := false
	for h := abi.ChainEpoch(1); h < 3*testTick; h++ {
		plan := PlanActorBalanceWrite(h, testTicks, testInterval, true, true)

		if plan.Mode == ActorBalanceSkip {
			continue
		}
		if plan.Mode == ActorBalanceFull {
			t.Fatalf("height=%d: production cadence must never write a full row-set (zero hour=%v)", h, common.IsZeroHour(true, h))
		}
		if plan.Mode != ActorBalanceChanged {
			t.Fatalf("height=%d: unexpected mode %s", h, plan.Mode)
		}
		if plan.PrevHeight >= h {
			t.Fatalf("height=%d: prev %d must be strictly below", h, plan.PrevHeight)
		}

		if common.IsZeroHour(true, h) {
			sawZeroHour = true
			if plan.Mode.String() != "changed" {
				t.Fatalf("height=%d: zero hour must report the same shape as a tick, got %q", h, plan.Mode.String())
			}
		}
	}

	if !sawZeroHour {
		t.Fatal("no zero hour was exercised in three days")
	}
}

// TestPlanActorBalanceWritePreservesGate pins the planner to the exact gate
// expression it replaced in extractActorBalance:
//
//	(!IsZeroHour && !IsExtract) || !EnableExtractActorBalance  ->  skip
//
// and to the new rule that replaced "the zero hour always yields a full
// snapshot": the zero hour is a snapshot epoch like any other and must never
// write a full row-set while a delta base exists.
func TestPlanActorBalanceWritePreservesGate(t *testing.T) {
	ctx := &Ctx{Opts: Options{StateRegular: ActorStateRegularOptions{Interval: testInterval}}}

	for _, enabled := range []bool{true, false} {
		for h := abi.ChainEpoch(0); h < 2*testTick; h++ {
			zeroHour := common.IsZeroHour(true, h)
			proceedLegacy := (zeroHour || IsExtract(testTicks, ctx, h)) && enabled

			plan := PlanActorBalanceWrite(h, testTicks, testInterval, true, enabled)
			if (plan.Mode != ActorBalanceSkip) != proceedLegacy {
				t.Fatalf("enabled=%v height=%d: mode=%s, legacy gate proceeds=%v", enabled, h, plan.Mode, proceedLegacy)
			}
			if !enabled && plan.Mode != ActorBalanceSkip {
				t.Fatalf("height=%d: disabled must skip, got %s", h, plan.Mode)
			}
			if zeroHour && enabled && plan.Mode == ActorBalanceFull {
				t.Fatalf("height=%d: the zero hour must not write a full snapshot any more, got %s", h, plan.Mode)
			}
		}
	}
}

func testCode(t *testing.T, seed string) cid.Cid {
	t.Helper()

	h, err := multihash.Sum([]byte(seed), multihash.SHA2_256, -1)
	if err != nil {
		t.Fatalf("multihash: %v", err)
	}

	return cid.NewCidV1(cid.Raw, h)
}

// TestActorBalanceNeedsWrite is the "only write on change" predicate:
// Balance/Code changes (including first appearance and removal) count,
// everything else (Head/Nonce churn) must not.
func TestActorBalanceNeedsWrite(t *testing.T) {
	account := testCode(t, "account")
	miner := testCode(t, "storageminer")
	headA := testCode(t, "head-a")
	headB := testCode(t, "head-b")

	actor := func(code cid.Cid, head cid.Cid, nonce uint64, balance int64) *types.Actor {
		return &types.Actor{Code: code, Head: head, Nonce: nonce, Balance: big.NewInt(balance)}
	}

	cases := []struct {
		name string
		old  *types.Actor
		cur  *types.Actor
		want bool
	}{
		{name: "unchanged must not be written", old: actor(account, headA, 1, 100), cur: actor(account, headA, 1, 100), want: false},
		{name: "head+nonce churn only must not be written", old: actor(account, headA, 1, 100), cur: actor(account, headB, 7, 100), want: false},
		{name: "balance changed must be written", old: actor(account, headA, 1, 100), cur: actor(account, headB, 2, 101), want: true},
		{name: "balance drained to zero must be written", old: actor(account, headA, 1, 100), cur: actor(account, headA, 1, 0), want: true},
		{name: "code changed must be written", old: actor(miner, headA, 1, 100), cur: actor(account, headA, 1, 100), want: true},
		{name: "first appearance must be written", old: nil, cur: actor(account, headA, 1, 100), want: true},
		{name: "removal must be written (tombstone)", old: actor(account, headA, 1, 100), cur: nil, want: true},
		{name: "both absent must not be written", old: nil, cur: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ActorBalanceNeedsWrite(tc.old, tc.cur); got != tc.want {
				t.Fatalf("ActorBalanceNeedsWrite = %v, want %v", got, tc.want)
			}
		})
	}
}
