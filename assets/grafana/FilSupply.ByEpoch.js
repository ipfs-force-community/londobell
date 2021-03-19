[
	{
		$match: {
			_id: {
				$gte: ctx.From,
				$lte: ctx.To,
			},
		},
	},
	{
		$sort: {
			_id: 1
		}
	},
	{
		$set: {
			Values: {
				$objectToArray: "$$ROOT.CirculatingSupply"
			}
		}
	},
	{
		$set: {
			"Values.Epoch": "$_id",
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
						1e18,
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
