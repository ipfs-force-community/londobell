package common

// richListTopAggregator is the richlist read against the daily top-N snapshot
// collection (model.ActorBalanceTop) instead of ActorBalance.
//
// Same predicate and same shape as the upstream richlist.js — the code list and
// the positive-balance filter are unchanged, so the response is the same top
// 1000 — with one addition: a secondary sort on Addr. The writer ranks equal
// balances by address ascending, and pinning that tiebreak here makes the
// endpoint's ordering deterministic instead of "whatever mongo returns".
//
// The upstream version is still a valid fallback: ActorBalanceTop carries the
// same field names (Epoch, Code, Balance, Addr), so if the override is switched
// off the upstream pipeline renders against the same collection and returns the
// same rows (only without the stable tiebreak).
//
// The $in list must stay equal to extract.RichListCodes; TestRichListTopPipeline
// pins the two together.
var richListTopAggregator = `
[
    {
        $match: {
            "Epoch": ctx.StartEpoch,
            "Code" : {$in:["storageminer","account","multisig"]},
            "Balance": {$gt: "0"},
        }
    },
    {
        $addFields: {
            balance: {$toDecimal: "$Balance"}
        }
    },
    {
        $sort: {
            "balance": -1,
            "Addr": 1,
        }
    },
    {
        $limit: 1000
    },
    {
        $skip: ctx.Skip
    },
    {
        $limit: ctx.Limit
    }
]
`

// RichListTopPipeline returns the local richlist pipeline when overrides are
// enabled, otherwise the upstream (ActorBalance-shaped) version.
func RichListTopPipeline(upstream []byte) []byte {
	if pipelineV2Enabled() {
		return []byte(richListTopAggregator)
	}

	return upstream
}
