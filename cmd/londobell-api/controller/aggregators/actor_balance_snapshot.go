package aggregators

import (
	"github.com/filecoin-project/go-state-types/abi"

	rextract "github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

// actorBalanceSnapshotOptions returns the ActorBalance snapshot cadence the
// writer uses in production: regular ticks every ActorBalanceTicks * interval
// epochs and a full snapshot at each local zero hour. It is read straight from
// the extractor defaults instead of being duplicated here, so the writer and
// these readers can never drift apart.
func actorBalanceSnapshotOptions() (ticks int, interval abi.ChainEpoch, zeroHour bool) {
	opts := rextract.DefaultOptions()

	return opts.StateRegular.ActorBalanceTicks, opts.StateRegular.Interval, opts.ZeroHourExtract.ActorBalance
}

// latestActorBalanceFullSnapshotEpoch returns the most recent epoch <= epoch
// that carries a FULL ActorBalance row-set (the zero hour).
//
// ActorBalance is written in full only at the zero hour; every regular tick
// writes just the actors whose Balance/Code changed. So:
//
//   - GetRichList must query this epoch: it needs the whole actor set at one
//     epoch, and a forward-fill would silently drop every actor that did not
//     move that tick.
//   - GetBalance uses it as the lower bound of its forward-fill window, which
//     guarantees every actor has a row inside [fullSnapshot, requestedEpoch].
//
// ok is false when epoch predates the first snapshot (epoch < first zero
// hour); callers then fall back to querying the requested epoch alone, which is
// the pre-optimisation behaviour.
func latestActorBalanceFullSnapshotEpoch(epoch int64) (int64, bool) {
	ticks, interval, zeroHour := actorBalanceSnapshotOptions()

	full, ok := rextract.LatestFullSnapshotHeight(abi.ChainEpoch(epoch), ticks, interval, zeroHour)
	if !ok {
		return epoch, false
	}

	return int64(full), true
}
