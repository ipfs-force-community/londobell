[
    {
        $match: {
            $and: [
                {"Depth": 1},
                {"Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}},
                {$or: [{"Msg.From":{$regex: /^1/}}, {"Msg.From":{$regex: /^3/}}, {"Msg.From":{$regex: /^4/}}]},
                {"Msg.Method": {$gt: 0}}
            ]
        }
    },
    {
        $lookup:
            {
                from: "Message",
                let: {cid: "$Cid"},
                pipeline: [
                    {
                        $match:
                            {
                                $expr: {
                                    $and: [
                                        {$eq: [ "$_id", "$$cid"]},
                                        {$not: {$regexMatch: {input: "$Detail.Actor", regex: "multisig"}}}
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
        $group: {
            _id:0,
            Count: {$sum:1}
        }
    }
]