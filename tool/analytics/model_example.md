## Example
### AllocatedSectors

#### allocated-sectors

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"Count": 0,
		"RawBytes": 0,
		"RunCount": 0,
		"Runs": [
			{
				"Len": 0,
				"Val": false
			}
		]
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### BlockHeader

#### block-header

```
{
	"ElectionProof": {
		"VRFProof": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		},
		"WinCount": 0
	},
	"Epoch": 0,
	"Messages": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"Miner": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### ClaimedPower

#### claimed-power-v2

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
		"SealProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### claimed-power-v3

```
{
	"Addr": "073366",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### ExecGas

#### exec-gas

```
{
	"Charges": [
		{
			"C": 0,
			"Callers": [
				1073741824
			],
			"Name": "",
			"S": 0,
			"VC": 0,
			"VS": 0
		}
	],
	"Epoch": 0,
	"_id": ""
}
```


### ExecTrace

#### exec-trace: multisig.ProposeReturn v3

```
{
	"Cid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Depth": 0,
	"Detail": {
		"Return": {
			"Applied": false,
			"Code": 0,
			"Ret": {
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			},
			"TxnID": 0
		}
	},
	"Epoch": 0,
	"Error": "",
	"GasCost": {
		"BaseFeeBurn": "1024",
		"GasUsed": "340282591298641078465964189926313473653",
		"Message": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"MinerPenalty": "340282591298641078465964189926313473653",
		"MinerTip": "1024",
		"OverEstimationBurn": "1073741824",
		"Refund": "1073741824",
		"TotalCost": "340282591298641078465964189926313473653"
	},
	"Msg": {
		"From": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"Method": 2,
		"To": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q"
	},
	"MsgRct": {
		"ExitCode": 0,
		"GasUsed": 0,
		"Return": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		}
	},
	"Seq": [
		0
	],
	"SeqIndex": [
		[
			0
		]
	],
	"SubCallCount": 0,
	"Ver": "",
	"_id": ""
}
```


### FilSupply

#### fil-supply

```
{
	"CirculatingSupply": {
		"FilBurnt": "340282591298641078465964189926313473653",
		"FilCirculating": "1073741824",
		"FilLocked": "1024",
		"FilMined": "1073741824",
		"FilVested": "1024"
	},
	"_id": 0
}
```


### Message

#### message: miner.PreCommitSector v2

```
{
	"Detail": {
		"Actor": "fil/2/storageminer",
		"Method": "PreCommitSector",
		"Params": {
			"DealIDs": [
				0
			],
			"Expiration": 0,
			"ReplaceCapacity": false,
			"ReplaceSectorDeadline": 0,
			"ReplaceSectorNumber": 0,
			"ReplaceSectorPartition": 0,
			"SealProof": 0,
			"SealRandEpoch": 0,
			"SealedCID": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
			"SectorNumber": 0
		}
	},
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### message: miner.PreCommitSector v3

```
{
	"Detail": {
		"Actor": "fil/3/storageminer",
		"Method": "PreCommitSector",
		"Params": {
			"DealIDs": [
				0
			],
			"Expiration": 0,
			"ReplaceCapacity": false,
			"ReplaceSectorDeadline": 0,
			"ReplaceSectorNumber": 0,
			"ReplaceSectorPartition": 0,
			"SealProof": 0,
			"SealRandEpoch": 0,
			"SealedCID": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
			"SectorNumber": 0
		}
	},
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### message: multisig.Propose v2

```
{
	"Detail": {
		"Actor": "fil/2/multisig",
		"Method": "Propose",
		"Params": {
			"Method": 0,
			"Params": {
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			},
			"To": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
			"Value": "340282591298641078465964189926313473653"
		}
	},
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### message: multisig.Propose v3

```
{
	"Detail": {
		"Actor": "fil/3/multisig",
		"Method": "Propose",
		"Params": {
			"Method": 0,
			"Params": {
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			},
			"To": "073366",
			"Value": "1024"
		}
	},
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### MinerFunds

#### miner-funds

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"FeeDebt": "1073741824",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "1073741824",
		"VestingTotal": "340282591298641078465964189926313473653"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### MinerSectorSummary

#### miner-sector-summary

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"Summaries": [
			{
				"DealCount": 0,
				"LowerBound": 0,
				"SectorCount": 0,
				"TotalDealWeight": "1024",
				"TotalInitialPledge": "340282591298641078465964189926313473653",
				"TotalVerifiedDealWeight": "1073741824",
				"UpperBound": 0
			}
		]
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### MiningProfitability

#### mining-profitability

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"ExpectedDayReward": "1024",
		"InitialConsensusPledge": "340282591298641078465964189926313473653",
		"InitialPledge": "1073741824",
		"InitialStoragePledge": "1024",
		"ProjectionOfFaultFee": "340282591298641078465964189926313473653",
		"ProjectionOfInitialPledge": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### MultisigBalance

#### multisig-balance

```
{
	"Addr": "073366",
	"Detail": {
		"Locked": "1024",
		"Vested": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### Tipset

#### tipset

```
{
	"BaseFee": "1024",
	"ChildEpoch": 0,
	"Cids": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"MinTimestamp": 0,
	"Receipts": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"State": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Weight": "340282591298641078465964189926313473653",
	"_id": 0
}
```


### VerifiedRegistry

#### verifreg

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"Cap": "1073741824",
		"Type": ""
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


