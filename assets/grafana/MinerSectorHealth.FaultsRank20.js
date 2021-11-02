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
            "Values.Addr": "$Addr"
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
            "k": "FaultsQAPower"
        }
    },
    {
        $project: {
            _id: "$Addr",
            Value: {
                $toDouble: {
                    $divide: [
                        {
                            $toDecimal: "$v",

                        },
                        1024 * 1024 * 1024 * 1024 * 1024,
                    ]
                }
            },
            Epoch: 1,
        }
    },
    {
        $sort: {
            Value:  - 1
        }
    },
    {
        $group: {
            _id: "$Epoch",
            Value: {
                $push: {
                    Value: "$Value",
                    addr: "$_id"
                }
            },

        }
    },
    {
        $project: {
            Value: {
                $slice: ["$Value", 20]
            }
        }
    },
    {
        $unwind: "$Value",
    },
    {
        $set: {
            "Value.Epoch": "$_id",
            "Value._id": "$Value.addr"
        }
    },
    {
        $replaceRoot: {
            newRoot: "$Value",
        }
    },
    {
        $group: {
            _id: "$addr",
            points: {
                $push: "$$ROOT",
            },
        }
    },
]