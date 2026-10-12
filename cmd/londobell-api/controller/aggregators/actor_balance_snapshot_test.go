package aggregators

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"

	"github.com/ipfs-force-community/londobell/common"
	rextract "github.com/ipfs-force-community/londobell/racailum/segment/extract"
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
// richlist would query a delta tick and get a near-empty Top1000.
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

// TestLatestActorBalanceFullSnapshotEpoch is the richlist epoch arithmetic:
// it must always land on a zero hour (a full row-set), never on a delta tick.
func TestLatestActorBalanceFullSnapshotEpoch(t *testing.T) {
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
			got, ok := latestActorBalanceFullSnapshotEpoch(tc.epoch)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("epoch = %d, got %d, want %d", tc.epoch, got, tc.want)
			}
			if got > tc.epoch {
				t.Fatalf("full snapshot %d must not be above the requested epoch %d", got, tc.epoch)
			}

			// With zero-hour snapshots on, the chosen epoch must be a genuine
			// full snapshot and never a plain delta tick.
			ticks, interval, zeroHour := actorBalanceSnapshotOptions()
			if tc.wantOK && zeroHour {
				if !common.IsZeroHour(true, abi.ChainEpoch(got)) {
					t.Fatalf("epoch %d is not a zero hour", got)
				}
				if tick := int64(ticks) * int64(interval); tick > 0 && got%tick == 0 {
					t.Fatalf("epoch %d is a delta tick, not a full snapshot", got)
				}
			}
		})
	}
}
