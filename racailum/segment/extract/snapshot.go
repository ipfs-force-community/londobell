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
// snapshots. The zero-hour snapshot stays full (the readers anchor on it);
// the regular ticks now only write the actors whose Balance or Code changed
// since the previous snapshot.
//
// Invariant the readers rely on: every ActorBalance row-set written between
// two consecutive full snapshots is a *delta* on top of the last full
// snapshot. A reader that forward-fills (Epoch <= X, latest row wins) is
// therefore correct, but the richlist reader (which needs one epoch with the
// *whole* set) must query a full-snapshot epoch.

// epochsPerDay is the number of Filecoin epochs in a local day:
// 24h * 60m * 60s / 30s. The zero-hour cadence repeats on this period.
const epochsPerDay = abi.ChainEpoch(24 * 60 * 60 / 30)

// ActorBalanceMode is the write strategy picked for one ActorBalance snapshot
// epoch.
type ActorBalanceMode int

const (
	// ActorBalanceSkip: this epoch is not a snapshot epoch at all.
	ActorBalanceSkip ActorBalanceMode = iota
	// ActorBalanceFull: write every actor (the zero-hour snapshot).
	ActorBalanceFull
	// ActorBalanceChanged: write only the actors whose Balance/Code changed
	// since PrevHeight.
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
//   - zero hour -> full (byte-for-byte the legacy snapshot)
//   - not a regular tick -> skip
//   - no determinable previous snapshot, or the previous snapshot is a whole
//     tick interval (or more) back -> full. A whole-interval gap means the
//     previous row-set is itself a delta with no recent full snapshot behind
//     it, so there is no base to diff against; also, with zero-hour
//     extraction switched off there is no full snapshot at all, and this
//     falls back to the legacy "full every tick" behaviour.
//   - otherwise -> changed, diffing against PrevHeight.
func PlanActorBalanceWrite(height abi.ChainEpoch, ticks int, interval abi.ChainEpoch, zeroHour, enabled bool) ActorBalanceWritePlan {
	if !enabled {
		return ActorBalanceWritePlan{Mode: ActorBalanceSkip}
	}

	if common.IsZeroHour(zeroHour, height) {
		return ActorBalanceWritePlan{Mode: ActorBalanceFull}
	}

	if !IsTickEpoch(ticks, interval, height) {
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

// LatestFullSnapshotHeight returns the greatest epoch <= height that carries a
// FULL ActorBalance row-set: a zero hour while zero-hour extraction is on.
//
// The richlist reader needs exactly this epoch (a forward-fill would mix a
// delta into a "whole set at one epoch" query and silently drop actors).
//
// With zero-hour extraction off there is no full snapshot, so this degrades to
// the latest tick epoch, which is precisely where PlanActorBalanceWrite falls
// back to full writes in that configuration.
func LatestFullSnapshotHeight(height abi.ChainEpoch, ticks int, interval abi.ChainEpoch, zeroHour bool) (abi.ChainEpoch, bool) {
	if height < 0 {
		return 0, false
	}

	if zeroHour {
		return latestZeroHourAtOrBefore(height)
	}

	if tick := TickInterval(ticks, interval); tick > 0 {
		return height - height%tick, true
	}

	return height, true
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
