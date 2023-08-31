## Schema
### ActorAddress

```
{
	"DelegatedAddress": "address.Address",
	"Epoch": "abi.ChainEpoch",
	"RobustAddress": "address.Address",
	"_id": "address.Address"
}
```

### ActorBalance

```
{
	"Addr": "address.Address",
	"Addresses": "[]address.Address",
	"Balance": "big.Int",
	"Code": "string",
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### ActorEvent

```
{
	"ActorID": "address.Address",
	"Cid": "cid.Cid",
	"Data": "ethtypes.EthBytes",
	"Epoch": "abi.ChainEpoch",
	"LogIndex": "uint64",
	"Removed": "bool",
	"SignedCid": "cid.Cid",
	"Topics": "[]ethtypes.EthHash",
	"_id": "string"
}
```

### ActorMessage

```
{
	"ActorID": "address.Address",
	"Cid": "cid.Cid",
	"Epoch": "abi.ChainEpoch",
	"ExitCode": "exitcode.ExitCode",
	"From": "address.Address",
	"IsBlock": "bool",
	"MethodName": "string",
	"SignedCid": "cid.Cid",
	"To": "address.Address",
	"TransferType": "string",
	"Type": "string",
	"Value": "big.Int",
	"_id": "string"
}
```

### ActorState

```
{
	"Addr": "address.Address",
	"Balance": "big.Int",
	"Code": "string",
	"Detail": "model.ActorStateDetail",
	"Epoch": "abi.ChainEpoch",
	"_id": "cid.Cid"
}
```

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

### Allocations

```
{
	"AllocationID": "verifreg.AllocationId",
	"Client": "abi.ActorID",
	"Data": "cid.Cid",
	"Epoch": "abi.ChainEpoch",
	"Expiration": "abi.ChainEpoch",
	"Provider": "abi.ActorID",
	"Size": "abi.PaddedPieceSize",
	"TermMax": "abi.ChainEpoch",
	"TermMin": "abi.ChainEpoch",
	"_id": "string"
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
	"MessageCount": "int",
	"Messages": "cid.Cid",
	"Miner": "address.Address",
	"Ticket": {
		"VRFProof": "[]uint8"
	},
	"_id": "cid.Cid"
}
```

### BlockMessage

```
{
	"Epoch": "abi.ChainEpoch",
	"Messages": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### ChangedActor

```
{
	"ActorID": "address.Address",
	"Address": "*address.Address",
	"Balance": "big.Int",
	"Code": "string",
	"Epoch": "abi.ChainEpoch",
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

### Claims

```
{
	"ClaimID": "uint64",
	"Client": "abi.ActorID",
	"Data": "cid.Cid",
	"Epoch": "abi.ChainEpoch",
	"Provider": "abi.ActorID",
	"Sector": "abi.SectorNumber",
	"Size": "abi.PaddedPieceSize",
	"TermMax": "abi.ChainEpoch",
	"TermMin": "abi.ChainEpoch",
	"TermStart": "abi.ChainEpoch",
	"_id": "string"
}
```

### DatacapAllowances

```
{
	"Amount": "big.Int",
	"Epoch": "abi.ChainEpoch",
	"Operator": "abi.ActorID",
	"Owner": "abi.ActorID",
	"_id": "cid.Cid"
}
```

### DatacapBalances

```
{
	"Amount": "big.Int",
	"Epoch": "abi.ChainEpoch",
	"Owner": "abi.ActorID",
	"_id": "cid.Cid"
}
```

### DealProposal

```
{
	"Client": "address.Address",
	"ClientCollateral": "big.Int",
	"ClientID": "address.Address",
	"EndEpoch": "abi.ChainEpoch",
	"Epoch": "abi.ChainEpoch",
	"Label": "string",
	"PieceCID": "cid.Cid",
	"PieceSize": "abi.PaddedPieceSize",
	"Provider": "address.Address",
	"ProviderCollateral": "big.Int",
	"ProviderID": "address.Address",
	"StartEpoch": "abi.ChainEpoch",
	"StoragePricePerEpoch": "big.Int",
	"VerifiedDeal": "bool",
	"_id": "int64"
}
```

### DealProposalDetail

```
{
	"ActorStateExBasic": {
		"Addr": "address.Address",
		"Epoch": "abi.ChainEpoch",
		"Path": "[]cid.Cid",
		"_id": "cid.Cid"
	},
	"Detail": {
		"UnVerifiedDealCount": "uint64",
		"UnVerifiedDealEndCount": "uint64",
		"VerifiedDealCount": "uint64",
		"VerifiedDealEndCount": "uint64"
	}
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

### EthHash

```
{
	"Cid": "cid.Cid",
	"Epoch": "abi.ChainEpoch",
	"_id": "ethtypes.EthHash"
}
```

### EventsRoot

```
{
	"Epoch": "abi.ChainEpoch",
	"Events": "[]uint8",
	"_id": "cid.Cid"
}
```

### EvmInitCode

```
{
	"Epoch": "abi.ChainEpoch",
	"InitCode": "string",
	"_id": "address.Address"
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
	"IsBlock": "bool",
	"Msg": {
		"From": "address.Address",
		"Method": "abi.MethodNum",
		"MethodName": "string",
		"To": "address.Address",
		"Value": "big.Int"
	},
	"MsgRct": {
		"EventsRoot": "*cid.Cid",
		"ExitCode": "exitcode.ExitCode",
		"GasUsed": "int64",
		"Return": "[]uint8"
	},
	"Seq": "[]int",
	"SeqIndex": "[][]int",
	"SignedCid": "cid.Cid",
	"SubCallCount": "int",
	"Ver": "string",
	"_id": "string"
}
```

### ExplicitMessage

```
{
	"Epoch": "abi.ChainEpoch",
	"ExitCode": "exitcode.ExitCode",
	"From": "address.Address",
	"MethodName": "string",
	"To": "address.Address",
	"Value": "big.Int",
	"_id": "cid.Cid"
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
		"FilReserveDisbursed": "big.Int",
		"FilVested": "big.Int"
	},
	"_id": "abi.ChainEpoch"
}
```

### FinalHeight

```
{
	"Cids": "[]cid.Cid",
	"_id": "abi.ChainEpoch"
}
```

### MarketFunds

```
{
	"Addr": "address.Address",
	"Detail": {
		"ClientUnLockCollateralInFuture": "[]big.Int",
		"ClientUnlockStorageFeeInFuture": "[]big.Int",
		"ProviderUnLockCollateralInFuture": "[]big.Int",
		"TotalClientLockedCollateral": "big.Int",
		"TotalClientStorageFee": "big.Int",
		"TotalLocked": "big.Int",
		"TotalProviderLockedCollateral": "big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### Message

```
{
	"Detail": {
		"Actor": "string",
		"Method": "string",
		"PackedHeight": "abi.ChainEpoch",
		"Params": "model.MessageParams"
	},
	"SignedCid": "cid.Cid",
	"_id": "cid.Cid"
}
```

### MinerDealSector

```
{
	"DealIDs": "[]abi.DealID",
	"DealWeight": "big.Int",
	"Epoch": "abi.ChainEpoch",
	"InitialPledge": "big.Int",
	"Miner": "address.Address",
	"QAPower": "big.Int",
	"SealProof": "abi.RegisteredSealProof",
	"SectorNumber": "abi.SectorNumber",
	"VerifiedDealWeight": "big.Int",
	"_id": "string"
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
		"PledgeRelease": "[]big.Int",
		"PreCommitDeposits": "big.Int",
		"VestInFuture": "[]big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Info": {
		"AvailableBalance": "big.Int",
		"Balance": "big.Int",
		"Beneficiary": "address.Address",
		"BeneficiaryTerm": {
			"Expiration": "abi.ChainEpoch",
			"Quota": "big.Int",
			"UsedQuota": "big.Int"
		},
		"ConsensusFaultElapsed": "abi.ChainEpoch",
		"ControlAddresses": "[]address.Address",
		"FeeDebt": "big.Int",
		"Multiaddrs": "[][]uint8",
		"Owner": "address.Address",
		"PeerID": "[]uint8",
		"PendingBeneficiaryTerm": {
			"ApprovedByBeneficiary": "bool",
			"ApprovedByNominee": "bool",
			"NewBeneficiary": "address.Address",
			"NewExpiration": "abi.ChainEpoch",
			"NewQuota": "big.Int"
		},
		"PendingOwnerAddress": "*address.Address",
		"PendingWorkerKey": {
			"EffectiveAt": "abi.ChainEpoch",
			"NewWorker": "address.Address"
		},
		"PrecommitSectorCount": "uint64",
		"SectorSize": "abi.SectorSize",
		"State": null,
		"WindowPoStPartitionSectors": "uint64",
		"WindowPoStProofType": "abi.RegisteredPoStProof",
		"Worker": "address.Address"
	},
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### MinerSector

```
{
	"Activation": "abi.ChainEpoch",
	"DealIDs": "[]abi.DealID",
	"DealWeight": "big.Int",
	"Epoch": "abi.ChainEpoch",
	"Expiration": "abi.ChainEpoch",
	"InitialPledge": "big.Int",
	"Miner": "address.Address",
	"SectorNumber": "abi.SectorNumber",
	"SimpleQaPower": "bool",
	"Terminated": "bool",
	"VerifiedDealWeight": "big.Int",
	"_id": "string"
}
```

### MinerSectorSummary

```
{
	"Addr": "address.Address",
	"Detail": {
		"CommittedCapacity": "uint64",
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
		"Mined": "big.Int",
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
		"VestInFuture": "[]big.Int",
		"Vested": "big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### NewDealProposal

```
{
	"Client": "address.Address",
	"ClientCollateral": "big.Int",
	"ClientID": "address.Address",
	"EndEpoch": "abi.ChainEpoch",
	"Epoch": "abi.ChainEpoch",
	"Label": {},
	"PieceCID": "cid.Cid",
	"PieceSize": "abi.PaddedPieceSize",
	"Provider": "address.Address",
	"ProviderCollateral": "big.Int",
	"ProviderID": "address.Address",
	"StartEpoch": "abi.ChainEpoch",
	"StoragePricePerEpoch": "big.Int",
	"VerifiedDeal": "bool",
	"_id": "abi.DealID"
}
```

### PendingTxns

```
{
	"Addr": "address.Address",
	"Detail": {
		"Approved": "[]address.Address",
		"Params": "[]uint8",
		"To": "address.Address",
		"TxnID": "int64",
		"Value": "big.Int"
	},
	"Epoch": "abi.ChainEpoch",
	"Path": "[]cid.Cid",
	"_id": "cid.Cid"
}
```

### SectorClaim

```
{
	"Client": "abi.ActorID",
	"Data": "cid.Cid",
	"Epoch": "abi.ChainEpoch",
	"Provider": "abi.ActorID",
	"Sector": "abi.SectorNumber",
	"Size": "abi.PaddedPieceSize",
	"TermMax": "abi.ChainEpoch",
	"TermMin": "abi.ChainEpoch",
	"TermStart": "abi.ChainEpoch",
	"_id": "uint64"
}
```

### StateFinalHeight

```
{
	"Cids": "[]cid.Cid",
	"_id": "abi.ChainEpoch"
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

