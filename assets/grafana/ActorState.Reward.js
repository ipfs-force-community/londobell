[
	{
		$match: {
			Epoch: {
				$gte: ctx.From,
				$lte: ctx.To,
			},
			Code: {
				$in: [
					"fil/2/reward",
					"fil/3/reward",
				]
			}
		},
	},
	{
		$sort: {
			Epoch: 1
		}
	},
	{
		$project: {
			Epoch: 1,
			Values: [
				{
					_id: "PerEpochReward",
					Epoch: "$Epoch",
					Value: {
						$toDouble: {
							$divide: [
								{
									$toDecimal: "$Detail.ThisEpochReward",
								},
								1e18
							]
						},
					}
				},
			]
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
