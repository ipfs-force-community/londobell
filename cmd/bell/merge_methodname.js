[
    {
        $match: {
            // "IsBlock":true,
            "Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
        }
    },
    {
        $lookup: {
            from: "Message",
            localField: "Cid",
            foreignField: "_id",
            as: "message",
        }
    },
    {
        $unwind: "$message"
    },
    {
        $addFields: {
            "Msg.From": "$message.From",
            "Msg.To":"$message.To",
            "Msg.Value": "$message.Value",
            "Msg.Method":"$message.Method",
            "Msg.MethodName": "$message.Detail.Method"
        }
    },
    {
        $project: {
            _id: 1,
            "Msg.From":1,
            "Msg.To":1,
            "Msg.Value": 1,
            "Msg.Method":1,
            "Msg.MethodName":1
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


