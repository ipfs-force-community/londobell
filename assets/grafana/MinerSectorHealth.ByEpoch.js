[
    {
        $match: {
            Epoch: {
                $gte: ctx.From,
                $lte: ctx.To,
            },
        },
    },
    {
        $set: {
            Values: {
                $objectToArray: "$$ROOT.Detail"
            }
        }
    },
    {
        $set: {
            "Values.Epoch": "$Epoch",
        }
    },
    {
        $unwind: "$Values",
    },
    {
        $replaceRoot: {
            newRoot: "$Values",
        }
    },
    {
        $match: {
            $or :[{"k" : "UnprovenQAPower"},{"k" : "RecoveriesQAPower"},{"k" : "FaultsQAPower"},{"k" : "ActiveSectorsQAPower"}]
        }
    },
    {
        $project: {
            _id: "$k",
            Value: {
                $toDouble: {
                    $divide: [
                        {
                            $toDecimal: "$v",
                        },
                        1024*1024*1024*1024*1024,
                    ]
                }
            },
            Epoch: 1,
        }
    },
    {
        $group: {
            _id: {
                "_id" : "$_id",
                "Epoch":"$Epoch"
            },
            Value: { $sum:"$Value"},
        }
    },
    {
        $set: {
            "Values.Epoch": "$_id.Epoch",
            "Values._id":"$_id._id",
            "Values.Value":"$Value"
        }
    },
    {
        $unwind: "$Values",
    },
    {
        $replaceRoot: {
            newRoot: "$Values",
        }
    },
    {
        $group: {
            _id: "$_id",
            points: {
                $push: "$$ROOT",
            }
        }
    }
]