package extract

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

// This file holds the *pure* ActorBalance snapshot arithmetic shared by the
// writer (racailum/segment/extract/tipset) and the readers
// (cmd/londobell-api). It deliberately depends on nothing but abi + the
// zero-hour helper so both sides can be unit-tested without a chain, a
// database or a network.
//
// Background: bell.ActorBalance is the biggest write hotspot on mainnet
// (1.706e9 documents / 589.5GB). Every tick used to write a full snapshot
// (~3.84e6 rows) while only ~0.1% of actors actually change between two
// snapshots.
//
// 2026-10-12: the daily full snapshot is gone. The zero hour used to write
// every actor; it now writes only the actors whose Balance or Code changed
// since the previous snapshot, exactly like a regular tick — the zero hour is
// still a *snapshot* epoch, it is just no longer a *full* one. No ActorBalance
// epoch carries the whole actor set any more. The richlist's "one epoch with
// the whole set" need is served by a separate, tiny daily top-N collection
// (model.ActorBalanceTop) built during the very same state-tree traversal.
//
// Invariant the readers rely on: every ActorBalance row-set is a *delta* on top
// of the row-set written at the previous snapshot epoch. A reader that
// forward-fills (scan back from X, newest row wins) is therefore correct.
// ActorBalanceFull survives only as the re-anchor path: when there is no
// determinable base (first snapshot, or a snapshot epoch was skipped) the
// writer still emits the whole set so a delta chain never starts from nothing.

// epochsPerDay is the number of Filecoin epochs in a local day:
// 24h * 60m * 60s / 30s. The zero-hour cadence repeats on this period.
const epochsPerDay = abi.ChainEpoch(24 * 60 * 60 / 30)

// ActorBalanceMode is the write strategy picked for one ActorBalance snapshot
// epoch.
type ActorBalanceMode int

const (
	// ActorBalanceSkip: this epoch is not a snapshot epoch at all.
	ActorBalanceSkip ActorBalanceMode = iota
	// ActorBalanceFull: write every actor. Only used when there is no delta
	// base to diff against, so the delta chain gets a starting row-set.
	ActorBalanceFull
	// ActorBalanceChanged: write only the actors whose Balance/Code changed
	// since PrevHeight. This is now the mode of *every* snapshot epoch,
	// including the zero hour.
	ActorBalanceChanged
)

func (m ActorBalanceMode) String() string {
	switch m {
	case ActorBalanceFull:
		return "full"
	case ActorBalanceChanged:
		return "changed"
	default:
		return "skip"
	}
}

// TickInterval is the regular snapshot cadence in epochs (ticks * interval).
// It returns 0 when ticks <= 0, which means "every epoch is a tick".
func TickInterval(ticks int, interval abi.ChainEpoch) abi.ChainEpoch {
	if ticks <= 0 {
		return 0
	}

	return abi.ChainEpoch(ticks) * interval
}

// IsTickEpoch reports whether height is a regular tick epoch.
//
// It is the ctx-free twin of IsExtract(tickOption, ctx, height) with
// interval = ctx.Opts.StateRegular.Interval: tickOption <= 0 means "every
// epoch". TestIsTickEpochMatchesIsExtract pins the two together.
func IsTickEpoch(ticks int, interval abi.ChainEpoch, height abi.ChainEpoch) bool {
	tick := TickInterval(ticks, interval)
	if tick <= 0 {
		return true
	}

	return height%tick == 0
}

// IsSnapshotEpoch reports whether an ActorBalance row-set is written at
// height at all (zero hour or regular tick).
func IsSnapshotEpoch(ticks int, interval abi.ChainEpoch, zeroHour bool, height abi.ChainEpoch) bool {
	return common.IsZeroHour(zeroHour, height) || IsTickEpoch(ticks, interval, height)
}

// latestZeroHourAtOrBefore returns the greatest epoch <= height that is a
// local midnight. zero-hour epochs repeat every epochsPerDay, so at most one
// day has to be scanned. ok is false when no zero hour exists at/below height.
func latestZeroHourAtOrBefore(height abi.ChainEpoch) (abi.ChainEpoch, bool) {
	if height < 0 {
		return 0, false
	}

	low := height - epochsPerDay + 1
	if low < 0 {
		low = 0
	}

	for h := height; h >= low; h-- {
		if common.IsZeroHour(true, h) {
			return h, true
		}
	}

	return 0, false
}

// PreviousSnapshotHeight returns the greatest epoch strictly below height at
// which an ActorBalance row-set is written, i.e. the base the "changed" mode
// diffs against.
//
// ok is false when no previous snapshot can be determined (first snapshot /
// height too small); callers must fall back to a full write in that case.
func PreviousSnapshotHeight(height abi.ChainEpoch, ticks int, interval abi.ChainEpoch, zeroHour bool) (abi.ChainEpoch, bool) {
	if height <= 0 {
		return 0, false
	}

	cand := height - 1

	var best abi.ChainEpoch
	if IsTickEpoch(ticks, interval, cand) {
		best = cand
	} else {
		tick := TickInterval(ticks, interval)
		best = cand - cand%tick
	}

	if zeroHour {
		if zh, ok := latestZeroHourAtOrBefore(cand); ok && zh > best {
			best = zh
		}
	}

	if best < 0 {
		return 0, false
	}

	return best, true
}

// ActorBalanceWritePlan is the pure decision for one ActorBalance snapshot
// epoch. PrevHeight is only meaningful when Mode == ActorBalanceChanged.
type ActorBalanceWritePlan struct {
	Mode       ActorBalanceMode
	PrevHeight abi.ChainEpoch
}

// PlanActorBalanceWrite decides what (if anything) to write for one epoch.
//
// Rules, in order:
//   - extraction disabled -> skip
//   - neither the zero hour nor a regular tick -> skip
//   - no determinable previous snapshot, or the previous snapshot is a whole
//     tick interval (or more) back -> full. A whole-interval gap means the
//     previous row-set is itself a delta with no recent base behind it, so
//     there is no base to diff against; also, with zero-hour extraction
//     switched off there is no base at all and this falls back to the legacy
//     "full every tick" behaviour.
//   - otherwise -> changed, diffing against PrevHeight.
//
// The zero hour has no special case left: it is a snapshot epoch like any
// other and writes only the actors that changed (2026-10-12).
func PlanActorBalanceWrite(height abi.ChainEpoch, ticks int, interval abi.ChainEpoch, zeroHour, enabled bool) ActorBalanceWritePlan {
	if !enabled {
		return ActorBalanceWritePlan{Mode: ActorBalanceSkip}
	}

	if !IsSnapshotEpoch(ticks, interval, zeroHour, height) {
		return ActorBalanceWritePlan{Mode: ActorBalanceSkip}
	}

	prev, ok := PreviousSnapshotHeight(height, ticks, interval, zeroHour)
	if !ok {
		return ActorBalanceWritePlan{Mode: ActorBalanceFull}
	}

	if tick := TickInterval(ticks, interval); tick > 0 && height-prev >= tick {
		return ActorBalanceWritePlan{Mode: ActorBalanceFull}
	}

	return ActorBalanceWritePlan{Mode: ActorBalanceChanged, PrevHeight: prev}
}

// IsTopSnapshotEpoch reports whether the daily top-N richlist snapshot is
// written at height: the local midnight, when zero-hour ActorBalance extraction
// is enabled. The top-N snapshot is strictly daily (one row-set per day), which
// is why this is the successor of "the zero hour has a full ActorBalance
// row-set" — that is no longer true, so the richlist's anchor moved here.
func IsTopSnapshotEpoch(zeroHour bool, height abi.ChainEpoch) bool {
	return common.IsZeroHour(zeroHour, height)
}

// EffectiveActorBalanceTopN returns the configured daily top-N size, falling
// back to DefaultActorBalanceTopN when unset (<= 0). A config file written
// before the field existed decodes to 0, and 0 must mean "the default", never
// "write nothing".
func EffectiveActorBalanceTopN(n int) int {
	if n > 0 {
		return n
	}

	return DefaultActorBalanceTopN
}

// LatestTopSnapshotHeight returns the greatest epoch <= height that carries a
// daily top-N richlist snapshot, i.e. the latest local midnight.
//
// This replaced LatestFullSnapshotHeight: no ActorBalance epoch carries a full
// actor set any more, so the richlist anchors on the top-N collection instead.
// It takes no tick/interval because the top-N cadence does not depend on them.
//
// ok is false when no top-N snapshot can exist at/below height (zero-hour
// extraction off, or height below the first midnight); callers then fall back
// to querying the requested epoch alone.
func LatestTopSnapshotHeight(height abi.ChainEpoch, zeroHour bool) (abi.ChainEpoch, bool) {
	if height < 0 {
		return 0, false
	}

	if !zeroHour {
		return 0, false
	}

	return latestZeroHourAtOrBefore(height)
}

// ActorBalanceNeedsWrite is the "only write on change" predicate.
//
// old is the actor as of the previous snapshot (nil when it did not exist
// then), cur is the actor as of the current snapshot (nil when it has been
// removed). Only Balance and Code count: other Actor fields (Head, Nonce)
// churn every epoch and would defeat the whole optimisation.
//
// A removed actor still needs a row (with a zero balance) so the readers'
// forward-fill can tombstone it instead of returning a stale balance forever.
func ActorBalanceNeedsWrite(old, cur *types.Actor) bool {
	switch {
	case cur == nil:
		return old != nil
	case old == nil:
		return true
	default:
		return !cur.Balance.Equals(old.Balance) || cur.Code != old.Code
	}
}
