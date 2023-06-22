[
    {
        $match: {
            "IsBlock": true,
            "Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
        }
    },
    {
        $project: {
            _id: 0,
            EventsRoot: "$MsgRct.EventsRoot",
            Epoch: "$Epoch"
        }
    }
]

