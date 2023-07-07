// DealProposal
[
    {
        $match: {
            _id: ctx.ID,
        }
    },
    {
        $project: {
            _id: 1,
            ProviderID: ctx.ProviderID,
            ClientID: ctx.ClientID
        }
    },
    {
        $merge: {
            into: "DealProposal",
            on: "_id",
            whenMatched:   "merge",
            whenNotMatched: "discard"
        }
    }
]