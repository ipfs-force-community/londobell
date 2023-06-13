// Tipset
[
    {
        $match: {
            _id: ctx.StartEpoch
        }
    },
    {
        $project: {
            Cids: "$Cids"
        }
    }
]