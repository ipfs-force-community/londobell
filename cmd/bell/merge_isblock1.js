[
    {
        $match: {
            "Depth": 1,
            "Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
            "Msg.From": /^([14])/
        }
    },
    {
        $addFields: {
            isBlock: true
        }
    },
    {
        $project: {
            _id: 1,
            IsBlock:"$isBlock",
        }
    },
    {
        $merge: {
            into: "ExecTrace",
            on: "_id",
            whenMatched:   "merge",
            whenNotMatched: "discard"
        }
    }
]

