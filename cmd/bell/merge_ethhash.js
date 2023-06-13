// ExecTrace
[
    {
        $match: {
            IsBlock: true,
            "Msg.From": /^4/,
            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}
        }
    },
    {
        $project: {
            _id: 0,
            Cid: {
                $cond: {
                    if:{
                        $eq:["$SignedCid", null]
                    }, then: "$Cid",
                    else: "$SignedCid"
                }
            },
            Epoch: "$Epoch"
        }
    }
]