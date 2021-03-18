[
	{
		$match: {
			Epoch: {
				$gte: ctx.From,
				$lte: ctx.To,
			},
		}
	},
	{
		$sort: {
			Epoch: 1
		}
	},
	{
		$project: {
			_id: 0,
			Epoch: 1,
			"Methods._id": 1,
			"Methods.GasUsed": 1,
			"Total.GasUsed": 1,
		}
	},
	{
		$set: {
			"Total._id": {
				"Epoch" : "$Epoch",
				"Actor" : "",
				"Method" : "Total"
			}
		}
	},
	{
		$set: {
			"Methods": {
				$concatArrays: [
					"$Methods",
					["$Total"],
				]
			}
		}
	},
	{
		$project: {
			Methods: 1,
		}
	},
	{
		$unwind: "$Methods",
	},
	{
		$group: {
			_id: {
				$concat: ["$Methods._id.Actor", "-", "$Methods._id.Method"]
			},
			points: {
				$push: {
					Epoch: "$$ROOT.Methods._id.Epoch",
					Value: "$$ROOT.Methods.GasUsed",
				},
			}
		}
	}
]
