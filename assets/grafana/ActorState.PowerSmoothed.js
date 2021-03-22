[
	{
		$match: {
			Epoch: {
				$gte: ctx.From,
				$lte: ctx.To,
			},
			Code: {
				$in: [
					"fil/2/storagepower",
					"fil/3/storagepower",
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
					_id: "QAPowerSmoothedPositionEstimate",
					Epoch: "$Epoch",
					Value: {
						$toDouble: {
							$divide: [
								{
									$toDecimal: "$Detail.ThisEpochQAPowerSmoothed.PositionEstimate",
								},
								1e48
							]
						},
					}
				},
				{
					_id: "QAPowerSmoothedVelocityEstimate",
					Epoch: "$Epoch",
					Value: {
						$toDouble: {
							$divide: [
								{
									$toDecimal: "$Detail.ThisEpochQAPowerSmoothed.VelocityEstimate"
								},
								1e48
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
