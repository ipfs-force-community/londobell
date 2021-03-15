[
	{
		$match: {
			Epoch: ctx.Epoch,
			Depth: 1,
			"GasCost.GasUsed": {
				$ne: null
			}
		}
	},
	{
		$project: {
			Cid: 1,
			Epoch: 1,
			GasUsed: {$toLong: "$GasCost.GasUsed"},
			BaseFeeBurn: {$toDecimal: "$GasCost.BaseFeeBurn"},
			OverEstimationBurn: {$toDecimal: "$GasCost.OverEstimationBurn"},
			MinerTip: {$toLong: "$GasCost.MinerTip"},
			MinerPenalty: {$toLong: "$GasCost.MinerPenalty"},
		}
	},
	{
		$lookup: {
			from: "Message",
			let: {
				msgID: "$Cid",
			},
			pipeline: [
				{
					$match: {
						$expr: {
							$eq: ["$_id", "$$msgID"],
						}
					}
				},
				{
					$project: {
						_id: 0,
						"Detail.Actor": 1,
						"Detail.Method": 1,
						GasLimit: 1,
						GasPremium: {$toLong: "$GasPremium"},
					}
				}
			],
			as: "RawMsg"
		}
	},
	{
		$unwind: "$RawMsg",
	},
	{
		$group: {
			_id: {
				Epoch: "$Epoch",
				Actor: "$RawMsg.Detail.Actor",
				Method: "$RawMsg.Detail.Method",
			},
			GasUsed: {$sum: "$GasUsed"},
			GasLimit: {$sum: "$RawMsg.GasLimit"},
			GasPremium: {$sum: "$RawMsg.GasPremium"},
			BaseFeeBurn: {$sum: "$BaseFeeBurn"},
			OverEstimationBurn: {$sum: "$OverEstimationBurn"},
			MinerPenalty: {$sum: "$MinerPenalty"},
			MsgCount: {$sum: 1},
		}
	},
	{
		$group: {
			_id: "$_id.Epoch",
			Methods: {
				$push: "$$ROOT",
			}
		}
	},
	{
		$addFields: {
			Total: {
				GasUsed: {$sum: "$Methods.GasUsed"},
				GasLimit: {$sum: "$Methods.GasLimit"},
				GasPremium: {$sum: "$Methods.GasPremium"},
				BaseFeeBurn: {$sum: "$Methods.BaseFeeBurn"},
				OverEstimationBurn: {$sum: "$Methods.OverEstimationBurn"},
				MinerPenalty: {$sum: "$Methods.MinerPenalty"},
				MsgCount: {$sum: "$Methods.MsgCount"},
			},
			ParentBaseFee: {$toDecimal: ctx.ParentBaseFee},
			Epoch: ctx.Epoch,
			MinTimestamp: ctx.MinTimestamp,
		}
	},
	{
		$merge: {
			into: "GasMonitoring",
			on: "_id",
			whenMatched: "keepExisting",
		},
	}
]
