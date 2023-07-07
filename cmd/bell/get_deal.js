// DealProposal
[
    {$match: {
        Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
        }},
    {
        $project: {
            Epoch: 1,
            Client:1,
            Provider:1
        }
    }
]