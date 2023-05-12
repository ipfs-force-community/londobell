[
    {
        $match: {
            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}
        }
    },
    {
        $project: {
            Epoch: "$Epoch",
            Cid: "$Cid",
            SignedCid: "$SignedCid",
            Value: "$Msg.Value",
            MethodName: "$Msg.MethodName",
            ExitCode: "$MsgRct.ExitCode",
            From: "$Msg.From",
            To: "$Msg.To",
            IsBlock: "$IsBlock"
        }
    }
]