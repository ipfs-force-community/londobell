## Schema
### AllocatedSectors

```
{
	"Addr": "address.Address",
	"Detail": {
		"Count": "uint64",
		"RawBytes": "int",
		"RunCount": "int",
		"Runs": "[]rlepluslazy.Run"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### BlockHeader

```
{
	"ElectionProof": {
		"VRFProof": "[]uint8",
		"WinCount": "int64"
	},
	"Epoch": "abi.ChainEpoch",
	"Messages": "cid.Cid",
	"Miner": "address.Address",
	"_id": "cid.Cid"
}
```

### ClaimedPower

```
{
	"Addr": "address.Address",
	"Detail": "model.ClaimedPowerDetail",
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### DealProposalSummary

```
{
	"ActorStateExBasic": {
		"Addr": "address.Address",
		"Epoch": "abi.ChainEpoch",
		"Path": "[]cid.Cid",
		"_id": "cid.Cid"
	},
	"Detail": {
		"Regular": {
			"ClientCollateral": "big.Int",
			"Clients": "uint64",
			"Count": "uint64",
			"PieceSize": "big.Int",
			"ProviderCollateral": "big.Int",
			"Providers": "uint64"
		},
		"Verified": {
			"ClientCollateral": "big.Int",
			"Clients": "uint64",
			"Count": "uint64",
			"PieceSize": "big.Int",
			"ProviderCollateral": "big.Int",
			"Providers": "uint64"
		}
	}
}
```

### ExecGas

```
{
	"Charges": "[]common.GasTraceCompact",
	"Epoch": "abi.ChainEpoch",
	"_id": "string"
}
```

### ExecTrace

```
{
	"Cid": "cid.Cid",
	"Depth": "int",
	"Detail": {
		"Return": "model.ExecTraceReturn"
	},
	"Epoch": "abi.ChainEpoch",
	"Error": "string",
	"GasCost": {
		"BaseFeeBurn": "big.Int",
		"GasUsed": "big.Int",
		"Message": "cid.Cid",
		"MinerPenalty": "big.Int",
		"MinerTip": "big.Int",
		"OverEstimationBurn": "big.Int",
		"Refund": "big.Int",
		"TotalCost": "big.Int"
	},
	"Msg": {
		"From": "address.Address",
		"Method": "abi.MethodNum",
		"To": "address.Address"
	},
	"MsgRct": {
		"ExitCode": "exitcode.ExitCode",
		"GasUsed": "int64",
		"Return": "[]uint8"
	},
	"Seq": "[]int",
	"SeqIndex": "[][]int",
	"SubCallCount": "int",
	"Ver": "string",
	"_id": "string"
}
```

### FilSupply

```
{
	"CirculatingSupply": {
		"FilBurnt": "big.Int",
		"FilCirculating": "big.Int",
		"FilLocked": "big.Int",
		"FilMined": "big.Int",
		"FilVested": "big.Int"
	},
	"_id": "abi.ChainEpoch"
}
```

### Message

```
{
	"Detail": {
		"Actor": "string",
		"Method": "string",
		"Params": "model.MessageParams"
	},
	"_id": "cid.Cid"
}
```

### MinerFunds

```
{
	"Addr": "address.Address",
	"Detail": {
		"FeeDebt": "big.Int",
		"InitialPledge": "big.Int",
		"LockedFunds": "big.Int",
		"PreCommitDeposits": "big.Int",
		"VestingTotal": "big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### MinerSectorSummary

```
{
	"Addr": "address.Address",
	"Detail": {
		"Summaries": "[]*model.MinerSectorSummaryRange"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### MiningProfitability

```
{
	"Addr": "address.Address",
	"Detail": {
		"ExpectedDayReward": "big.Int",
		"InitialConsensusPledge": "big.Int",
		"InitialPledge": "big.Int",
		"InitialStoragePledge": "big.Int",
		"ProjectionOfFaultFee": "big.Int",
		"ProjectionOfInitialPledge": "big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### MultisigBalance

```
{
	"Addr": "address.Address",
	"Detail": {
		"Locked": "big.Int",
		"Vested": "big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### Tipset

```
{
	"BaseFee": "big.Int",
	"ChildEpoch": "abi.ChainEpoch",
	"Cids": "[]cid.Cid",
	"MinTimestamp": "uint64",
	"Receipts": "cid.Cid",
	"State": "cid.Cid",
	"Weight": "big.Int",
	"_id": "abi.ChainEpoch"
}
```

### VerifiedRegistry

```
{
	"Addr": "address.Address",
	"Detail": {
		"Cap": "big.Int",
		"Type": "string"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

