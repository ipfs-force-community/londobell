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
		$sort: {
			Epoch: 1
		}
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
		$project: {
			_id: "$k",
			Value: {
				$toDouble: {
					$divide: [
						{
							$toDecimal: "$v",
						},
						1e9,
					]
				}
			},
			Epoch: 1,
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
