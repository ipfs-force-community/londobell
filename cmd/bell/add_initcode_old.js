//ExecTrace
[
    {
        $match:{
            "IsBlock":false,
            // "Msg.MethodName":"CreateExternal",
            "Msg.From": "01",
            "Msg.Method":1,
            "MsgRct.ExitCode": 0,
            "Epoch":{$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}
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
                                    {$regexMatch: { input: "$Detail.Actor", regex: "evm"}},
                                ]
                            }
                        }
                }
            ],
            as: "message"
        }
    },
    {
        $unwind: "$message",
    },
    {
        $project:{
            "Epoch":"$Epoch",
            "ActorID":"$Msg.To",
            "InitCode":"$message.Detail.Params.Initcode"
        }
    }
]