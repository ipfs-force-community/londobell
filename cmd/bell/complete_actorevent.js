// ExecTrace
[
    {
        $match: {
            IsBlock: true,
            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
            "MsgRct.EventsRoot": {$ne: null}
        }
    },
    {
        $lookup: {
            from: "EventsRoot",
            let: {
                eventsRoot: "$MsgRct.EventsRoot",
            },
            pipeline: [
                {
                    $match: {
                        $expr: {
                            $eq: ["$_id", "$$eventsRoot"],
                        }
                    }
                },
            ],
            as: "root"
        }
    },
    {
        $unwind: "$root",
    },
    {
        $project: {
            Cid: "$Cid",
            SignedCid: "$SignedCid",
            Epoch: "$Epoch",
            Events: "$root.Events"
        }
    },
]