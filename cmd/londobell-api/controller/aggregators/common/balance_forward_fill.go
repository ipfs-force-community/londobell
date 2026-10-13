package common

// balanceForwardFillAggregator replaces the upstream single-epoch balance
// query. ActorBalance is no longer written in full on every tick: the zero
// hour writes the whole set and each regular tick only writes the actors whose
// Balance/Code changed. A point lookup on one epoch would therefore return
// nothing for the ~99.9% of actors that did not move that tick.
//
// Instead of a point lookup we take the latest row at or before the requested
// epoch inside the caller-supplied window [ctx.StartEpoch, ctx.EndEpoch)
// (the caller starts that window at the most recent full snapshot, so every
// actor is guaranteed to have a row inside it) and let the newest row win.
// Mongo runs one such query per shard and the caller keeps the highest Epoch,
// which is what makes the forward-fill correct across shard boundaries.
//
// The upstream pipeline (balance.js) is embedded in the external
// londobell-aggregators module; this override is gated by the same
// LONDOBELL_AGG_PIPELINE_V2 switch as the other local pipeline overrides.
var balanceForwardFillAggregator = `
[
    {
        $match: {
            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
            Addr: ctx.Addr
        }
    },
    {
        $sort: {
            Epoch: -1
        }
    },
    {
        $limit: 1
    }
]
`

// BalanceForwardFillPipeline returns the forward-filling balance pipeline when
// local overrides are enabled, otherwise the upstream single-epoch version.
func BalanceForwardFillPipeline(upstream []byte) []byte {
	if pipelineV2Enabled() {
		return []byte(balanceForwardFillAggregator)
	}

	return upstream
}
