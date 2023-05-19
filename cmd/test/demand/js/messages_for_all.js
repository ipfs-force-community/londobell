[
    {
        $match: {
            $and: [
                {"Depth": 1},
                {"Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}},
                {$or: [{"Msg.From":{$regex: /^1/}}, {"Msg.From":{$regex: /^3/}}, {"Msg.From":{$regex: /^4/}}]},
            ]
        }
    },
    {
        $group: {
            _id:0,
            Count: {$sum:1}
        }
    }
]