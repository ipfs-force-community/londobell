[
    {
        $match: {
            // "IsBlock": true,
            "Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
        }
    },
    {
        $project: {
            _id: 0,
            Cid: {$cond: {
                    if:{
                        $eq:["$SignedCid", null]
                    }, then: "$Cid",
                    else: "$SignedCid"
                }
            },
            EventsRoot: "$MsgRct.EventsRoot",
            Epoch: "$Epoch"
        }
    }
]

