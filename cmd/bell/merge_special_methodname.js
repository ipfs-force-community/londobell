[
    {
        $match: {
            "IsBlock":true,
            "Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
        }
    },
    {
        $lookup: {
            from: "Message",
            let: {cid: "$Cid"},
            pipeline: [
                {
                    $match:
                        {
                            $expr: {
                                $and: [
                                        {$eq: ["$_id", "$$cid"]},
                                    {$or:[
                                            {$and: [
                                                    {$regexMatch: { input: "$Detail.Actor", regex: "ethaccount"}},
                                                    {$gte: ["$Method", 16777216]}
                                                ]},
                                            {$and: [
                                                    {$regexMatch: { input: "$Detail.Actor", regex: "evm"}},
                                                    {$gt: ["$Method", 1023]},
                                                    {$ne: ["$Method", 3844450837]}
                                                ]},
                                            {$regexMatch: {input: "$message.Detail.Actor", regex: "placeholder"}}
                                        ]}
                                ]
                            }
                        }
                }
            ],
            as: "message"
        }
    },
    {
        $unwind: "$message"
    },
    {
        $addFields: {
            "Msg.From": "$message.From",
            "Msg.To": "$message.To",
            "Msg.Value": "$message.Value",
            "Msg.Method": "$message.Method",
            "Msg.MethodName": {
                $cond: {
                    if: {
                        $and: [
                            {$regexMatch: {input: "$message.Detail.Actor", regex: "ethaccount"}},
                            {$gte: ["$message.Method", 16777216]}
                        ]
                    }
                    , then: "Send(ethaccount)",
                    else: {
                        $cond: {
                            if: {
                                $and: [
                                    {$regexMatch: {input: "$message.Detail.Actor", regex: "evm"}},
                                    {$gt: ["$message.Method", 1023]},
                                    {$ne: ["$message.Method", 3844450837]}
                                ]
                            },
                            then: "HandleFileCoinMethod",
                            else: {
                                $cond: {
                                    if: {
                                        $regexMatch: {input: "$message.Detail.Actor", regex: "placeholder"}
                                    },
                                    then: "Send(placeholder)",
                                    else: "$message.Detail.Method"
                                }
                            }
                        }
                    }
                },
            },
        },
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


