[
    {
        $match: {
            $and: [
                {"Depth": 1},
                {"Epoch": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}},
                // {$or: [{"Msg.From":{$regex: /^1/}}, {"Msg.From":{$regex: /^3/}}, {"Msg.From":{$regex: /^4/}}]},
                {"Msg.Method": 3844450837}
            ]
        }
    },
    // {
    //     $lookup:
    //         {
    //             from: "Message",
    //             let: {cid: "$Cid"},
    //             pipeline: [
    //                 {
    //                     $match:
    //                         {
    //                             $expr: {
    //                                 $and: [
    //                                     {$eq: [ "$_id", "$$cid"]},
    //                                     {$regexMatch: {input: "$Detail.Actor", regex: "placeholder"}},
    //                                     {$regexMatch: {input: "$Detail.Actor", regex: "evm"}},
    //                                     {$regexMatch: {input: "$Detail.Actor", regex: "eam"}},
    //                                     {$regexMatch: {input: "$Detail.Actor", regex: "ethaccount"}},
    //                                 ]
    //                             }
    //                         }
    //                 }
    //             ],
    //             as: "message"
    //         }
    // },
    // {
    //     $unwind: "$message"
    // },
    {
        $group: {
            _id:0,
            Count: {$sum:1}
        }
    }
]