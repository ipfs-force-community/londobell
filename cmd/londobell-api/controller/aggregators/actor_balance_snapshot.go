package aggregators

import (
	"github.com/filecoin-project/go-state-types/abi"

	rextract "github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

// actorBalanceSnapshotOptions returns the ActorBalance snapshot cadence the
// writer uses in production: regular ticks every ActorBalanceTicks * interval
// epochs and a snapshot at each local zero hour. It is read straight from the
// extractor defaults instead of being duplicated here, so the writer and these
// readers can never drift apart.
func actorBalanceSnapshotOptions() (ticks int, interval abi.ChainEpoch, zeroHour bool) {
	opts := rextract.DefaultOptions()

	return opts.StateRegular.ActorBalanceTicks, opts.StateRegular.Interval, opts.ZeroHourExtract.ActorBalance
}

// actorBalanceTopSnapshotEpoch returns the most recent epoch <= epoch carrying a
// daily top-N richlist snapshot, i.e. the latest local midnight.
//
// ActorBalance no longer has any epoch with the whole actor set: the zero hour
// writes only the actors whose Balance/Code changed, exactly like a regular
// tick. The richlist's "whole set at one epoch" read therefore moved to the
// separate ActorBalanceTop collection, which is written once per day at the
// zero hour. This function is the anchor the richlist queries.
//
// ok is false when no top-N snapshot can exist at/below epoch (epoch < the
// first local midnight, or zero-hour extraction off); callers then fall back to
// querying the requested epoch alone.
func actorBalanceTopSnapshotEpoch(epoch int64) (int64, bool) {
	_, _, zeroHour := actorBalanceSnapshotOptions()

	top, ok := rextract.LatestTopSnapshotHeight(abi.ChainEpoch(epoch), zeroHour)
	if !ok {
		return epoch, false
	}

	return int64(top), true
}

// actorBalanceForwardFillStartEpoch is the lower bound of the forward-fill
// window for a single-address balance lookup.
//
// It is 0 — the beginning of the chain — and that is deliberate, not a
// placeholder. A forward-fill (latest row at or before the requested epoch
// wins) is only correct if the window reaches back to the actor's most recent
// write. While the zero hour still wrote every actor, the latest midnight
// bounded that window: every actor had a row there. It does not any more, so an
// actor that last changed a week ago has no row anywhere near the latest
// snapshot, and a one-day window would silently return nothing for it. Scanning
// from 0 is the only bound that cannot drop a row; the pipeline still carries
// its Epoch range, so it stays an index seek on Epoch_1_Addr_1 (sparse — an
// ActorBalance query without an Epoch condition would degrade to a full
// collection scan, see docs/cn/batch_failure_semantics.md §8.15.4).
//
// Cost note: the number of index seeks grows with how far back the actor's last
// change is, up to ~1 seek per snapshot epoch of history for an address that
// never moved (order of 1e3, ~1ms each). If that ever shows up in latency, the
// fix is a {Addr:1, Epoch:-1} index, not a shorter window.
func actorBalanceForwardFillStartEpoch() int64 {
	return 0
}
