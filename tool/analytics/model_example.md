## Example
### ActorAddress

#### actor-address

```
{
	"DelegatedAddress": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Epoch": 0,
	"RobustAddress": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"_id": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq"
}
```


### ActorBalance

#### actor-balance

```
{
	"Addr": "073366",
	"Addresses": [
		"1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq"
	],
	"Balance": "1073741824",
	"Code": "",
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### ActorEvent

#### actor-event

```
{
	"ActorID": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Cid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Data": "0x2c2ec67e3e1fea8e4a39601cb3a3cd44f5fa830d",
	"Epoch": 0,
	"LogIndex": 0,
	"Removed": false,
	"SignedCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"Topics": [
		"0xc3535fefdd76f2de7a843fa4defcecb26cbc2d5b7279f7939662ca75815117eb"
	],
	"_id": ""
}
```


### ActorMessage

#### actor-message

```
{
	"ActorID": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Cid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Epoch": 0,
	"ExitCode": 0,
	"From": "073366",
	"IsBlock": false,
	"MethodName": "",
	"RootCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"RootSignedCid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"To": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"TransferType": "",
	"Type": "",
	"Value": "340282591298641078465964189926313473653",
	"_id": ""
}
```


### ActorState

#### actor-state: miner.State v0

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1024",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledgeRequirement": "1024",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
		"PreCommittedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"PreCommittedSectorsExpiry": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"VestingFunds": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	},
	"Epoch": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### actor-state: miner.State v2

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "1073741824",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
		"PreCommittedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"PreCommittedSectorsExpiry": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"VestingFunds": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	},
	"Epoch": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### actor-state: miner.State v3

```
{
	"Addr": "073366",
	"Balance": "1024",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"FeeDebt": "1024",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
		"PreCommittedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"PreCommittedSectorsExpiry": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"VestingFunds": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	},
	"Epoch": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### actor-state: miner.State v4

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "340282591298641078465964189926313473653",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
		"PreCommittedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"PreCommittedSectorsExpiry": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"VestingFunds": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	},
	"Epoch": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### actor-state: miner.State v5

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1073741824",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
		"PreCommittedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"PreCommittedSectorsCleanUp": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"VestingFunds": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	},
	"Epoch": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### actor-state: miner.State v6

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "1024",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "1024",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
		"PreCommittedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"PreCommittedSectorsCleanUp": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"VestingFunds": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	},
	"Epoch": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### actor-state: miner.State v7

```
{
	"Addr": "073366",
	"Balance": "340282591298641078465964189926313473653",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
		"PreCommittedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"PreCommittedSectorsCleanUp": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"VestingFunds": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	},
	"Epoch": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### actor-state: miner.State v8

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "1073741824",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
		"PreCommittedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"PreCommittedSectorsCleanUp": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"VestingFunds": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	},
	"Epoch": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### actor-state: miner.State v9

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1024",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"FeeDebt": "1024",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
		"PreCommittedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"PreCommittedSectorsCleanUp": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"VestingFunds": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	},
	"Epoch": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### actor-state: miner.State v10

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "340282591298641078465964189926313473653",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
		"PreCommittedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"PreCommittedSectorsCleanUp": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"VestingFunds": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	},
	"Epoch": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### actor-state: miner.State v11

```
{
	"Addr": "073366",
	"Balance": "1073741824",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
		"PreCommittedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"PreCommittedSectorsCleanUp": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"VestingFunds": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	},
	"Epoch": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### actor-state: miner.State v12

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "1024",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"CurrentDeadline": 0,
		"DeadlineCronActive": false,
		"Deadlines": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "1024",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
		"PreCommittedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"PreCommittedSectorsCleanUp": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"ProvingPeriodStart": 0,
		"Sectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"VestingFunds": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	},
	"Epoch": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### AllocatedSectors

#### allocated-sectors

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
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
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### Allocations

#### allocations-v10

```
{
	"AllocationID": 0,
	"Client": 0,
	"Data": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Epoch": 0,
	"Expiration": 0,
	"Provider": 0,
	"Size": 0,
	"TermMax": 0,
	"TermMin": 0,
	"_id": ""
}
```

#### allocations-v9

```
{
	"AllocationID": 0,
	"Client": 0,
	"Data": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Epoch": 0,
	"Expiration": 0,
	"Provider": 0,
	"Size": 0,
	"TermMax": 0,
	"TermMin": 0,
	"_id": ""
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
	"MessageCount": 0,
	"Messages": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Miner": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Ticket": {
		"VRFProof": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		}
	},
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### BlockMessage

#### block-message

```
{
	"Epoch": 0,
	"Messages": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### ChangedActor

#### changed-actor

```
{
	"ActorID": "073366",
	"Address": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "340282591298641078465964189926313473653",
	"Code": "",
	"Epoch": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### ClaimedPower

#### claimed-power-v0

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### claimed-power-v10

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### claimed-power-v11

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### claimed-power-v12

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
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### claimed-power-v2

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
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
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### claimed-power-v4

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### claimed-power-v5

```
{
	"Addr": "073366",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### claimed-power-v6

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### claimed-power-v7

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### claimed-power-v8

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### claimed-power-v9

```
{
	"Addr": "073366",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
		"WindowPoStProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### Claims

#### claims-v10

```
{
	"ClaimID": 0,
	"Client": 0,
	"Data": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Epoch": 0,
	"Provider": 0,
	"Sector": 0,
	"Size": 0,
	"TermMax": 0,
	"TermMin": 0,
	"TermStart": 0,
	"_id": ""
}
```

#### claims-v9

```
{
	"ClaimID": 0,
	"Client": 0,
	"Data": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"Epoch": 0,
	"Provider": 0,
	"Sector": 0,
	"Size": 0,
	"TermMax": 0,
	"TermMin": 0,
	"TermStart": 0,
	"_id": ""
}
```


### CreateMessage

#### create-message

```
{
	"ActorID": "073366",
	"Caller": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Cid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Epoch": 0,
	"From": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"IsBlock": false,
	"MethodName": "",
	"RootCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"RootSignedCid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"To": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Value": "1024",
	"_id": ""
}
```


### DatacapAllowances

#### datacap-allowances-v10

```
{
	"Amount": "1073741824",
	"Epoch": 0,
	"Operator": 0,
	"Owner": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### datacap-allowances-v9

```
{
	"Amount": "340282591298641078465964189926313473653",
	"Epoch": 0,
	"Operator": 0,
	"Owner": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### DatacapBalances

#### datacap-balances-v10

```
{
	"Amount": "1024",
	"Epoch": 0,
	"Owner": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### datacap-balances-v9

```
{
	"Amount": "1073741824",
	"Epoch": 0,
	"Owner": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### DealProposal

#### deal-proposal-full

```
{
	"Client": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"ClientCollateral": "1073741824",
	"ClientID": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"EndEpoch": 0,
	"Epoch": 0,
	"Label": "",
	"PieceCID": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"PieceSize": 0,
	"Provider": "073366",
	"ProviderCollateral": "1024",
	"ProviderID": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"StartEpoch": 0,
	"StoragePricePerEpoch": "340282591298641078465964189926313473653",
	"VerifiedDeal": false,
	"_id": 0
}
```


### DealProposalDetail

#### deal-proposal-detail

```
{
	"ActorStateExBasic": {
		"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"Epoch": 0,
		"Path": [
			"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
		],
		"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	},
	"Detail": {
		"UnVerifiedDealCount": 0,
		"UnVerifiedDealEndCount": 0,
		"VerifiedDealCount": 0,
		"VerifiedDealEndCount": 0
	}
}
```


### DealProposalSummary

#### deal-proposal-summary

```
{
	"ActorStateExBasic": {
		"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
		"Epoch": 0,
		"Path": [
			"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
		],
		"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	},
	"Detail": {
		"Regular": {
			"ClientCollateral": "1073741824",
			"Clients": 0,
			"Count": 0,
			"PieceSize": "340282591298641078465964189926313473653",
			"ProviderCollateral": "1024",
			"Providers": 0
		},
		"Verified": {
			"ClientCollateral": "1073741824",
			"Clients": 0,
			"Count": 0,
			"PieceSize": "340282591298641078465964189926313473653",
			"ProviderCollateral": "1024",
			"Providers": 0
		}
	}
}
```


### EthHash

#### eth-hash

```
{
	"Cid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Epoch": 0,
	"_id": "0x9941603956a8fd47753cbbdb3be2ae35f3afd8af164ab76222b72904f9ba84b8"
}
```


### EventsRoot

#### events-root

```
{
	"Epoch": 0,
	"Events": {
		"$binary": {
			"base64": "SGVsbG8=",
			"subType": "00"
		}
	},
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### EvmInitCode

#### evm-initcode

```
{
	"Epoch": 0,
	"InitCode": "",
	"_id": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla"
}
```


### ExecGas

#### exec-gas

```
{
	"Charges": [
		{
			"C": 0,
			"Name": "",
			"S": 0,
			"TG": 0
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
	"Cid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
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
	"FIL": 0,
	"GasCost": {
		"BaseFeeBurn": "1073741824",
		"GasUsed": "1024",
		"Message": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"MinerPenalty": "1024",
		"MinerTip": "1073741824",
		"OverEstimationBurn": "340282591298641078465964189926313473653",
		"Refund": "340282591298641078465964189926313473653",
		"TotalCost": "1024"
	},
	"IsBlock": false,
	"Msg": {
		"From": "073366",
		"Method": 2,
		"MethodName": "",
		"To": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"Value": "340282591298641078465964189926313473653"
	},
	"MsgRct": {
		"EventsRoot": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"ExitCode": 0,
		"GasUsed": 0,
		"Return": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		},
		"ReturnCodec": 0
	},
	"RootCid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"RootSignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Seq": [
		0
	],
	"SeqIndex": [
		[
			0
		]
	],
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"SubCallCount": 0,
	"Ver": "",
	"_id": ""
}
```


### ExplicitMessage

#### explicit-message

```
{
	"Epoch": 0,
	"ExitCode": 0,
	"From": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"MethodName": "",
	"To": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Value": "1073741824",
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### FilSupply

#### fil-supply

```
{
	"CirculatingSupply": {
		"FilBurnt": "1073741824",
		"FilCirculating": "1024",
		"FilLocked": "340282591298641078465964189926313473653",
		"FilMined": "1024",
		"FilReserveDisbursed": "1073741824",
		"FilVested": "340282591298641078465964189926313473653"
	},
	"_id": 0
}
```


### FinalHeight

#### final-height

```
{
	"Cids": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": 0
}
```


### MarketFunds

#### market-funds

```
{
	"Addr": "073366",
	"Detail": {
		"ClientUnLockCollateralInFuture": [
			"1024"
		],
		"ClientUnlockStorageFeeInFuture": [
			"340282591298641078465964189926313473653"
		],
		"ProviderUnLockCollateralInFuture": [
			"1073741824"
		],
		"TotalClientLockedCollateral": "1024",
		"TotalClientStorageFee": "340282591298641078465964189926313473653",
		"TotalLocked": "340282591298641078465964189926313473653",
		"TotalProviderLockedCollateral": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### Message

#### message: miner.PreCommitSector v2

```
{
	"Detail": {
		"Actor": "fil/2/storageminer",
		"Method": "PreCommitSector",
		"PackedHeight": 0,
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
	"GasFeeCap": "1024",
	"GasPremium": "1073741824",
	"Nonce": 0,
	"SignedCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### message: miner.PreCommitSector v3

```
{
	"Detail": {
		"Actor": "fil/3/storageminer",
		"Method": "PreCommitSector",
		"PackedHeight": 0,
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
	"GasFeeCap": "340282591298641078465964189926313473653",
	"GasPremium": "1024",
	"Nonce": 0,
	"SignedCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### message: multisig.Propose v2

```
{
	"Detail": {
		"Actor": "fil/2/multisig",
		"Method": "Propose",
		"PackedHeight": 0,
		"Params": {
			"Method": 0,
			"Params": {
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			},
			"To": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
			"Value": "1073741824"
		}
	},
	"GasFeeCap": "340282591298641078465964189926313473653",
	"GasPremium": "1024",
	"Nonce": 0,
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### message: multisig.Propose v3

```
{
	"Detail": {
		"Actor": "fil/3/multisig",
		"Method": "Propose",
		"PackedHeight": 0,
		"Params": {
			"Method": 0,
			"Params": {
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			},
			"To": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
			"Value": "1073741824"
		}
	},
	"GasFeeCap": "340282591298641078465964189926313473653",
	"GasPremium": "1024",
	"Nonce": 0,
	"SignedCid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### MinerDealSector

#### miner-deal-sector

```
{
	"DealIDs": [
		0
	],
	"DealWeight": "1073741824",
	"Epoch": 0,
	"InitialPledge": "1024",
	"Miner": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"QAPower": "1073741824",
	"SealProof": 0,
	"SectorNumber": 0,
	"VerifiedDealWeight": "340282591298641078465964189926313473653",
	"_id": ""
}
```


### MinerFunds

#### miner-funds

```
{
	"Addr": "073366",
	"Detail": {
		"FeeDebt": "1073741824",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PledgeRelease": [
			"1073741824"
		],
		"PreCommitDeposits": "340282591298641078465964189926313473653",
		"VestInFuture": [
			"1024"
		]
	},
	"Epoch": 0,
	"Info": {
		"AvailableBalance": "1024",
		"Balance": "340282591298641078465964189926313473653",
		"Beneficiary": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"BeneficiaryTerm": {
			"Expiration": 0,
			"Quota": "340282591298641078465964189926313473653",
			"UsedQuota": "1024"
		},
		"ConsensusFaultElapsed": 0,
		"ControlAddresses": [
			"3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla"
		],
		"FeeDebt": "1073741824",
		"Multiaddrs": [
			{
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			}
		],
		"Owner": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"PeerID": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		},
		"PendingBeneficiaryTerm": {
			"ApprovedByBeneficiary": false,
			"ApprovedByNominee": false,
			"NewBeneficiary": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
			"NewExpiration": 0,
			"NewQuota": "1073741824"
		},
		"PendingOwnerAddress": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
		"PendingWorkerKey": {
			"EffectiveAt": 0,
			"NewWorker": "073366"
		},
		"PrecommitSectorCount": 0,
		"SectorSize": 0,
		"State": null,
		"WindowPoStPartitionSectors": 0,
		"WindowPoStProofType": 0,
		"Worker": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q"
	},
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### MinerSector

#### miner-sector

```
{
	"Activation": 0,
	"DealIDs": [
		0
	],
	"DealWeight": "340282591298641078465964189926313473653",
	"Epoch": 0,
	"Expiration": 0,
	"InitialPledge": "1073741824",
	"Miner": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"SectorNumber": 0,
	"SimpleQaPower": false,
	"Terminated": false,
	"VerifiedDealWeight": "1024",
	"_id": ""
}
```


### MinerSectorSummary

#### miner-sector-summary

```
{
	"Addr": "073366",
	"Detail": {
		"CommittedCapacity": 0,
		"Summaries": [
			{
				"DealCount": 0,
				"LowerBound": 0,
				"SectorCount": 0,
				"TotalDealWeight": "340282591298641078465964189926313473653",
				"TotalInitialPledge": "1073741824",
				"TotalV1InitialPledge": "340282591298641078465964189926313473653",
				"TotalVerifiedDealWeight": "1024",
				"UpperBound": 0,
				"V1SectorCount": 0
			}
		]
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### MiningProfitability

#### mining-profitability

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"ExpectedDayReward": "1024",
		"InitialConsensusPledge": "340282591298641078465964189926313473653",
		"InitialPledge": "1073741824",
		"InitialStoragePledge": "1024",
		"Mined": "1024",
		"ProjectionOfFaultFee": "340282591298641078465964189926313473653",
		"ProjectionOfInitialPledge": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### MultisigBalance

#### multisig-balance

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"Locked": "1073741824",
		"VestInFuture": [
			"1024"
		],
		"Vested": "340282591298641078465964189926313473653"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### NewDealProposal

#### new-deal-proposal

```
{
	"Client": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"ClientCollateral": "1024",
	"ClientID": "073366",
	"EndEpoch": 0,
	"Epoch": 0,
	"Label": {},
	"PieceCID": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"PieceSize": 0,
	"Provider": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"ProviderCollateral": "340282591298641078465964189926313473653",
	"ProviderID": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"StartEpoch": 0,
	"StoragePricePerEpoch": "1073741824",
	"VerifiedDeal": false,
	"_id": 0
}
```


### PendingTxns

#### pending-txns

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"Approved": [
			"1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq"
		],
		"Params": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		},
		"To": "073366",
		"TxnID": 0,
		"Value": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### SectorClaim

#### sector-claim

```
{
	"Client": 0,
	"Data": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Epoch": 0,
	"Provider": 0,
	"Sector": 0,
	"Size": 0,
	"TermMax": 0,
	"TermMin": 0,
	"TermStart": 0,
	"_id": 0
}
```


### StateFinalHeight

#### state-final-height

```
{
	"Cids": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": 0
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
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
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


