[
    {
        $match: {
            "Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
            "Msg.From": /^([02])/
        }
    },
    {
        $addFields: {
            isBlock: false
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
