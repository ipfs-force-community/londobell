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
		$set: {
			"Total._id": {
				"Epoch" : "$Epoch",
				"Actor" : "Total",
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
		$replaceRoot: {
			newRoot: "$Methods",
		}
	},
	{
		$project: {
			_id: {
				$concat: ["$_id.Actor", "-", "$_id.Method"],
			},
			Epoch: "$_id.Epoch",
			Value: ctx.Data.Value,
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
