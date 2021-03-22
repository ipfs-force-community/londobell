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
					_id: "ThisEpochRawBytePower",
					Epoch: "$Epoch",
					Value: {
						$toDouble: {
							$divide: [
								{
									$toDecimal: "$Detail.ThisEpochRawBytePower",
								},
								1073741824
							]
						},
					}
				},
				{
					_id: "ThisEpochQualityAdjPower",
					Epoch: "$Epoch",
					Value: {
						$toDouble: {
							$divide: [
								{
									$toDecimal: "$Detail.ThisEpochQualityAdjPower",
								},
								1073741824
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
