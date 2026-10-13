package aggregators

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"

	"github.com/ipfs-force-community/londobell/common"
	rextract "github.com/ipfs-force-community/londobell/racailum/segment/extract"
	rmodel "github.com/ipfs-force-community/londobell/racailum/segment/model"
)

// dayEpochs is one local day in Filecoin epochs (24*60*60/30). The zero-hour
// cadence repeats on this period.
const dayEpochs = int64(24 * 60 * 60 / 30)

// aggZeroHourResidue finds the smallest non-negative local midnight. Derived
// rather than hardcoded so the test stays honest under a different genesis time
// (calibnet build tag).
func aggZeroHourResidue(t *testing.T) int64 {
	t.Helper()

	for h := int64(0); h < dayEpochs; h++ {
		if common.IsZeroHour(true, abi.ChainEpoch(h)) {
			return h
		}
	}

	t.Fatalf("no zero hour found in %d epochs", dayEpochs)
	return 0
}

// TestActorBalanceSnapshotOptionsMatchWriterDefaults pins the reader to the
// writer's cadence. If this changes, the readers must be updated in lockstep or
// richlist would query an epoch the writer never wrote.
func TestActorBalanceSnapshotOptionsMatchWriterDefaults(t *testing.T) {
	ticks, interval, zeroHour := actorBalanceSnapshotOptions()

	opts := rextract.DefaultOptions()
	if ticks != opts.StateRegular.ActorBalanceTicks ||
		interval != opts.StateRegular.Interval ||
		zeroHour != opts.ZeroHourExtract.ActorBalance {
		t.Fatalf("options diverge from extractor defaults: ticks=%d interval=%d zeroHour=%v", ticks, interval, zeroHour)
	}

	if ticks != 24 || interval != abi.ChainEpoch(builtin.EpochsInHour) || !zeroHour {
		t.Fatalf("unexpected writer cadence: ticks=%d interval=%d zeroHour=%v (want 24 / %d / true)",
			ticks, interval, zeroHour, builtin.EpochsInHour)
	}
}

// TestLatestActorBalanceTopSnapshotEpoch is the richlist epoch arithmetic: it
// must land on a local midnight, where the daily top-N snapshot lives. There is
// no full ActorBalance row-set to anchor on any more (2026-10-12).
func TestLatestActorBalanceTopSnapshotEpoch(t *testing.T) {
	r := aggZeroHourResidue(t)
	t.Logf("zero hour residue = %d (mod %d)", r, dayEpochs)

	cases := []struct {
		name   string
		epoch  int64
		want   int64
		wantOK bool
	}{
		{name: "on the zero hour", epoch: r, want: r, wantOK: true},
		{name: "100 epochs after the zero hour", epoch: r + 100, want: r, wantOK: true},
		{name: "on the next zero hour", epoch: r + dayEpochs, want: r + dayEpochs, wantOK: true},
		{name: "real mainnet tick", epoch: 6372000, want: 6372000 - ((6372000 - r) % dayEpochs), wantOK: true},
		{name: "before the first zero hour falls back to the requested epoch", epoch: r - 1, want: r - 1, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := actorBalanceTopSnapshotEpoch(tc.epoch)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("epoch = %d, got %d, want %d", tc.epoch, got, tc.want)
			}
			if got > tc.epoch {
				t.Fatalf("top-N snapshot %d must not be above the requested epoch %d", got, tc.epoch)
			}

			// With zero-hour snapshots on, the chosen epoch must be a genuine
			// top-N epoch (a local midnight) and never a plain delta tick.
			_, _, zeroHour := actorBalanceSnapshotOptions()
			if tc.wantOK && zeroHour {
				if !rextract.IsTopSnapshotEpoch(true, abi.ChainEpoch(got)) {
					t.Fatalf("epoch %d is not a top-N snapshot epoch", got)
				}
				if !common.IsZeroHour(true, abi.ChainEpoch(got)) {
					t.Fatalf("epoch %d is not a zero hour", got)
				}
			}
		})
	}
}

// TestActorBalanceForwardFillStartCoversChain pins the forward-fill window
// lower bound at the beginning of the chain.
//
// While the zero hour still wrote every actor, the latest midnight bounded the
// window safely. It does not any more, so any bound above 0 can silently drop
// an actor whose last change is older than the bound. This test exists so a
// future "optimisation" of that bound has to argue with a failing test first.
func TestActorBalanceForwardFillStartCoversChain(t *testing.T) {
	if got := actorBalanceForwardFillStartEpoch(); got != 0 {
		t.Fatalf("forward-fill start = %d, want 0 (no epoch carries a full ActorBalance row-set any more)", got)
	}
}

// TestRichListReadsTheTopCollection pins the reader's table to the writer's
// collection constant. A literal here could drift from the model and point the
// endpoint at an empty collection (200 + zero rows, which looks like "no data"
// rather than a bug).
func TestRichListReadsTheTopCollection(t *testing.T) {
	if rmodel.ActorBalanceTopCollection != (&rmodel.ActorBalanceTop{}).CollectionName() {
		t.Fatalf("collection constant %q != model CollectionName %q",
			rmodel.ActorBalanceTopCollection, (&rmodel.ActorBalanceTop{}).CollectionName())
	}

	if rmodel.ActorBalanceTopCollection == (&rmodel.ActorBalance{}).CollectionName() {
		t.Fatal("the top-N collection must not be the ActorBalance collection")
	}
}
