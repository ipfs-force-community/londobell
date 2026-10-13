package aggregators

import (
	"testing"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

// testBalanceRanges builds a CountUtil slice out of {Start, End} pairs. End is
// the exclusive end of a shard's half-open [Start, End) range, exactly as
// GetEpochRange / refreshEpochRange report it.
func testBalanceRanges(ranges ...[2]int64) []multiquery.CountUtil {
	out := make([]multiquery.CountUtil, 0, len(ranges))
	for _, rg := range ranges {
		out = append(out, multiquery.CountUtil{Start: rg[0], End: rg[1]})
	}

	return out
}

// TestPlanBalanceQueryAnswerableRange is the table that pins the single-address
// balance endpoint's answerable range to the pre-2026-10-12 semantics:
//
//	① a future epoch (past the published top)     -> null
//	② exactly at the upper bound                  -> normal forward-fill
//	③ inside the range, snapshot / non-snapshot   -> unchanged (window still
//	                                                 starts at the chain origin)
//	④ an epoch before the earliest covered epoch  -> null
func TestPlanBalanceQueryAnswerableRange(t *testing.T) {
	residue := aggZeroHourResidue(t)

	// A "now"-ish published top: two local days past the first local midnight,
	// plus a bit — so that a snapshot epoch (a local midnight) and a
	// non-snapshot epoch both sit comfortably inside the covered range, under
	// whichever network this build targets.
	top := residue + 2*dayEpochs + 120
	snapshotEpoch := residue + 2*dayEpochs // a local midnight <= top
	deltaEpoch := snapshotEpoch + 100      // inside the range, not a snapshot epoch

	// The production shape: shards tile the epoch line up to the published top.
	// Deliberately unsorted — the bound must come from min/max, not from the
	// first element. The newest shard's End is exclusive, so it covers up to
	// top.
	published := testBalanceRanges(
		[2]int64{top + 1 - 7*dayEpochs, top + 1}, // newest shard (End exclusive)
		[2]int64{0, top + 1 - 8*dayEpochs},       // oldest shard
	)

	// An instance whose earliest shard starts late: nothing covers the dawn.
	lateStart := testBalanceRanges(
		[2]int64{residue + dayEpochs, top + 1},
	)

	cases := []struct {
		name       string
		startEpoch int64
		countUtils []multiquery.CountUtil
		wantAnswer bool
	}{
		{
			name:       "future epoch just past the published top returns null",
			startEpoch: top + 1,
			countUtils: published,
			wantAnswer: false,
		},
		{
			// The regression this guard restores: on mainnet the published top
			// was 6449706 and a request for 6450480 (the next local midnight,
			// ~774 epochs later) answered a balance carried forward from the
			// newest written epoch instead of null.
			name:       "future epoch a day past the published top returns null",
			startEpoch: top + dayEpochs,
			countUtils: published,
			wantAnswer: false,
		},
		{
			name:       "exactly at the upper bound still forward-fills",
			startEpoch: top,
			countUtils: published,
			wantAnswer: true,
		},
		{
			name:       "snapshot epoch inside the range is unchanged",
			startEpoch: snapshotEpoch,
			countUtils: published,
			wantAnswer: true,
		},
		{
			name:       "non-snapshot epoch inside the range is unchanged",
			startEpoch: deltaEpoch,
			countUtils: published,
			wantAnswer: true,
		},
		{
			name:       "epoch before the earliest covered epoch returns null",
			startEpoch: residue,
			countUtils: lateStart,
			wantAnswer: false,
		},
		{
			name:       "no databases at all returns null",
			startEpoch: top,
			countUtils: nil,
			wantAnswer: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := planBalanceQuery(tc.startEpoch, tc.countUtils)

			if plan.Answerable != tc.wantAnswer {
				t.Fatalf("start=%d: Answerable=%v, want %v", tc.startEpoch, plan.Answerable, tc.wantAnswer)
			}

			if !tc.wantAnswer {
				return
			}

			// Every answerable request is still forward-filled from the chain
			// origin and still reports under the requested epoch: the guard must
			// not re-introduce a narrow window (that is what silently dropped
			// actors on 2026-10-12).
			if plan.FromEpoch != actorBalanceForwardFillStartEpoch() {
				t.Fatalf("start=%d: FromEpoch=%d, want %d", tc.startEpoch, plan.FromEpoch, actorBalanceForwardFillStartEpoch())
			}
			if plan.FromEpoch != 0 {
				t.Fatalf("start=%d: FromEpoch=%d, want 0 (forward-fill reaches the chain origin)", tc.startEpoch, plan.FromEpoch)
			}
			if plan.ToEpoch != tc.startEpoch+1 {
				t.Fatalf("start=%d: ToEpoch=%d, want %d", tc.startEpoch, plan.ToEpoch, tc.startEpoch+1)
			}
		})
	}
}

// TestPlanBalanceQueryUpperBoundIsLastCoveredEpoch pins the bound's arithmetic:
// the answerable top is max(End)-1 across every shard (End is the exclusive end
// of a half-open range), taken as a maximum and independent of the order the
// shards come back in.
func TestPlanBalanceQueryUpperBoundIsLastCoveredEpoch(t *testing.T) {
	top := int64(2500)

	// Ascending, so the newest shard is last: a naive "first element" bound
	// would be wrong here.
	utils := testBalanceRanges(
		[2]int64{0, 1000},
		[2]int64{1000, 2000},
		[2]int64{2000, top + 1},
	)

	if plan := planBalanceQuery(top, utils); !plan.Answerable {
		t.Fatalf("start=%d (the last covered epoch) must be answerable", top)
	}
	if plan := planBalanceQuery(top+1, utils); plan.Answerable {
		t.Fatalf("start=%d (one past the last covered epoch) must not be answerable", top+1)
	}

	// Same shards, reversed: the decision must not depend on ordering.
	reversed := testBalanceRanges(
		[2]int64{2000, top + 1},
		[2]int64{1000, 2000},
		[2]int64{0, 1000},
	)
	if plan := planBalanceQuery(top, reversed); !plan.Answerable {
		t.Fatalf("start=%d must be answerable regardless of shard order", top)
	}
	if plan := planBalanceQuery(top+1, reversed); plan.Answerable {
		t.Fatalf("start=%d must not be answerable regardless of shard order", top+1)
	}
}
