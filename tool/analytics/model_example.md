## Example
### ActorBalance

#### actor-balance

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Addresses": [
		"2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q"
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


### ActorMessage

#### actor-message

```
{
	"ActorID": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Cid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Epoch": 0,
	"ExitCode": 0,
	"From": "073366",
	"IsBlock": false,
	"MethodName": "",
	"SignedCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"To": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
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
		"AllocatedSectors": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledgeRequirement": "1024",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
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

#### actor-state: miner.State v2

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "1073741824",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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

#### actor-state: miner.State v3

```
{
	"Addr": "073366",
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
		"FeeDebt": "1024",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "1073741824",
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

#### actor-state: miner.State v4

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "340282591298641078465964189926313473653",
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
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
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

#### actor-state: miner.State v5

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1073741824",
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
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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

#### actor-state: miner.State v6

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "1024",
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
		"FeeDebt": "1024",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
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

#### actor-state: miner.State v7

```
{
	"Addr": "073366",
	"Balance": "340282591298641078465964189926313473653",
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
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
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

#### actor-state: miner.State v8

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "1073741824",
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
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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

#### actor-state: miner.State v9

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1024",
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
		"FeeDebt": "1024",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
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

#### actor-state: miner.State v10

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "340282591298641078465964189926313473653",
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
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
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

#### actor-state: miner.State v11

```
{
	"Addr": "073366",
	"Balance": "1073741824",
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
		"FeeDebt": "1073741824",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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
	"Miner": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
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


### ClaimedPower

#### claimed-power-v0

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### claimed-power-v10

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
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### claimed-power-v11

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
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### claimed-power-v2

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024",
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
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
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
	"Addr": "073366",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
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
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024",
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
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
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
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
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
	"Addr": "073366",
	"Detail": {
		"QualityAdjPower": "1073741824",
		"RawBytePower": "1024",
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
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"QualityAdjPower": "1024",
		"RawBytePower": "340282591298641078465964189926313473653",
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


### DatacapAllowances

#### datacap-allowances-v10

```
{
	"Amount": "1073741824",
	"Epoch": 0,
	"Operator": 0,
	"Owner": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### datacap-allowances-v9

```
{
	"Amount": "340282591298641078465964189926313473653",
	"Epoch": 0,
	"Operator": 0,
	"Owner": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### DatacapBalances

#### datacap-balances-v10

```
{
	"Amount": "1024",
	"Epoch": 0,
	"Owner": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### datacap-balances-v9

```
{
	"Amount": "1073741824",
	"Epoch": 0,
	"Owner": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### DealProposal

#### deal-proposal-full

```
{
	"Client": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"ClientCollateral": "1073741824",
	"EndEpoch": 0,
	"Epoch": 0,
	"Label": "",
	"PieceCID": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"PieceSize": 0,
	"Provider": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"ProviderCollateral": "1024",
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
		"Addr": "073366",
		"Epoch": 0,
		"Path": [
			"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
		],
		"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
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
		"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"Epoch": 0,
		"Path": [
			"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
		],
		"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
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
	"Cid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Epoch": 0,
	"_id": "0xc3535fefdd76f2de7a843fa4defcecb26cbc2d5b7279f7939662ca75815117eb"
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
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
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
	"Cid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
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
		"BaseFeeBurn": "1073741824",
		"GasUsed": "1024",
		"Message": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"MinerPenalty": "1024",
		"MinerTip": "1073741824",
		"OverEstimationBurn": "340282591298641078465964189926313473653",
		"Refund": "340282591298641078465964189926313473653",
		"TotalCost": "1024"
	},
	"IsBlock": false,
	"Msg": {
		"From": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
		"Method": 2,
		"MethodName": "",
		"To": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
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
	"SignedCid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
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
		"FilBurnt": "1024",
		"FilCirculating": "340282591298641078465964189926313473653",
		"FilLocked": "1073741824",
		"FilMined": "340282591298641078465964189926313473653",
		"FilReserveDisbursed": "1024",
		"FilVested": "1073741824"
	},
	"_id": 0
}
```


### FinalHeight

#### final-height

```
{
	"Cids": [
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
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
			"340282591298641078465964189926313473653"
		],
		"ClientUnlockStorageFeeInFuture": [
			"1073741824"
		],
		"ProviderUnLockCollateralInFuture": [
			"1024"
		],
		"TotalClientLockedCollateral": "340282591298641078465964189926313473653",
		"TotalClientStorageFee": "1073741824",
		"TotalLocked": "1073741824",
		"TotalProviderLockedCollateral": "1024"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
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
			"SealedCID": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
			"SectorNumber": 0
		}
	},
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
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
			"SealedCID": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
			"SectorNumber": 0
		}
	},
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
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
			"Value": "340282591298641078465964189926313473653"
		}
	},
	"SignedCid": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
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
			"Value": "1024"
		}
	},
	"SignedCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
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
		"PendingOwnerAddress": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
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
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### MinerSectorSummary

#### miner-sector-summary

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
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
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### MiningProfitability

#### mining-profitability

```
{
	"Addr": "073366",
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
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### MultisigBalance

#### multisig-balance

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"Locked": "1073741824",
		"VestInFuture": [
			"1024"
		],
		"Vested": "340282591298641078465964189926313473653"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### PendingTxns

#### pending-txns

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Detail": {
		"Approved": [
			"073366"
		],
		"Params": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		},
		"To": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
		"TxnID": 0,
		"Value": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
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


