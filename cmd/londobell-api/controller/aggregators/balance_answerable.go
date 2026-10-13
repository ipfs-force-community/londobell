package aggregators

import (
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

// balanceQueryPlan is the pure decision the single-address balance reader makes
// before it touches a database: the forward-fill window it will query and
// whether the requested epoch is inside the range this instance can answer at
// all.
type balanceQueryPlan struct {
	// FromEpoch is the lower bound of the forward-fill window: the beginning of
	// the chain (see actorBalanceForwardFillStartEpoch). It must stay 0 — no
	// epoch carries a full ActorBalance row-set any more, so any bound above 0
	// can silently drop an actor whose last change is older than the bound.
	FromEpoch int64
	// ToEpoch is the exclusive upper end of the window: the requested epoch + 1.
	ToEpoch int64
	// Answerable is false when the requested epoch falls outside the epoch
	// ranges this instance's databases cover. The reader then answers data:null
	// (HTTP 200, no error) without querying.
	Answerable bool
}

// planBalanceQuery decides, purely, what a balance request for startEpoch
// should do given the epoch ranges countUtils reports for the databases this
// aggregator reads.
//
// The answerable range is the union of the shards' half-open [Start, End)
// ranges, i.e. [min(Start), max(End)-1] — the epochs some shard holds a row for:
//
//   - Below min(Start) no shard holds a row at all, so a forward-fill has
//     nothing to find and the endpoint answers null.
//   - Above max(End)-1 the epoch has not been published: answering there would
//     carry the newest written balance forward onto an epoch that has not
//     happened yet.
//
// Why the upper bound has to be stated here and is not "free" any more:
//
// The pre-change reader (b201e376) took the forward-fill window's lower bound
// from rextract.LatestFullSnapshotHeight(startEpoch) — the latest local midnight
// <= startEpoch. That doubled as an upper-bound guard by accident: once
// startEpoch reached the first local midnight above the published range, the
// whole window [midnight, startEpoch+1) lay above the data, the query returned
// nothing, and the endpoint answered data:null. For example, with the published
// top at 6449706 on mainnet (local midnights at ...6447600, 6450480...), a
// request for 6450480 got a window of [6450480, 6450481) and came back null,
// while 6447600 and 6448320 both got a window that reached real rows.
//
// On 2026-10-12 the lower bound was changed to 0 (actorBalanceForwardFillStartEpoch)
// because the zero hour stopped writing a full row-set, so a midnight no longer
// guarantees a row per actor and a narrower window silently dropped rows. The
// change was right for the lower bound, but the upper-bound guard was a side
// effect of the old lower bound and went with it: the window now spans the whole
// chain, so a request for a future epoch matches the newest written row and the
// endpoint answered a balance for an epoch that has not happened.
//
// This plan restores the bound explicitly, on the REQUEST epoch and independent
// of the window, so it neither depends on a full snapshot existing nor drops an
// actor whose last change predates a window bound.
func planBalanceQuery(startEpoch int64, countUtils []multiquery.CountUtil) balanceQueryPlan {
	plan := balanceQueryPlan{
		FromEpoch:  actorBalanceForwardFillStartEpoch(),
		ToEpoch:    startEpoch + 1,
		Answerable: false,
	}

	if len(countUtils) == 0 {
		// No database covers any epoch: nothing can be answered.
		return plan
	}

	lo := countUtils[0].Start
	hi := countUtils[0].End - 1
	for i := 1; i < len(countUtils); i++ {
		if countUtils[i].Start < lo {
			lo = countUtils[i].Start
		}
		if countUtils[i].End-1 > hi {
			hi = countUtils[i].End - 1
		}
	}

	plan.Answerable = startEpoch >= lo && startEpoch <= hi

	return plan
}
