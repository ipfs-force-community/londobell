// DealProposal
[
    {$match: {
        _id: {$gte: ctx.StartID, $lt: ctx.EndID},
        }},
    {
        $project: {
            Epoch: 1,
            Client:1,
            Provider:1
        }
    }
]