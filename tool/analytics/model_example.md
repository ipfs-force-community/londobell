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
	"SignedCid": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
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
		"AllocatedSectors": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"EarlyTerminations": {
			"$binary": {
				"base64": "UgAQIf/3P/n///0fyP/////+Rw==",
				"subType": "00"
			}
		},
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledgeRequirement": "1024",
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

#### actor-state: miner.State v2

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "1073741824",
	"Code": "",
	"Detail": {
		"AllocatedSectors": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"CurrentDeadline": 0,
		"Deadlines": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"EarlyTerminations": {
			"$binary": {
				"base64": "QA==",
				"subType": "00"
			}
		},
		"FeeDebt": "1073741824",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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

#### actor-state: miner.State v3

```
{
	"Addr": "073366",
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
		"FeeDebt": "1024",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "1073741824",
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

#### actor-state: miner.State v4

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "340282591298641078465964189926313473653",
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
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
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

#### actor-state: miner.State v5

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1073741824",
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
		"FeeDebt": "1073741824",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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

#### actor-state: miner.State v6

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "1024",
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
		"FeeDebt": "1024",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
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

#### actor-state: miner.State v7

```
{
	"Addr": "073366",
	"Balance": "340282591298641078465964189926313473653",
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
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
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

#### actor-state: miner.State v8

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "1073741824",
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
		"FeeDebt": "1073741824",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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

#### actor-state: miner.State v9

```
{
	"Addr": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Balance": "1024",
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
		"FeeDebt": "1024",
		"Info": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PreCommitDeposits": "1073741824",
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

#### actor-state: miner.State v10

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Balance": "340282591298641078465964189926313473653",
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
		"FeeDebt": "340282591298641078465964189926313473653",
		"Info": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"InitialPledge": "1024",
		"LockedFunds": "1073741824",
		"PreCommitDeposits": "1024",
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

#### actor-state: miner.State v11

```
{
	"Addr": "073366",
	"Balance": "1073741824",
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
		"FeeDebt": "1073741824",
		"Info": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
		"InitialPledge": "340282591298641078465964189926313473653",
		"LockedFunds": "1024",
		"PreCommitDeposits": "340282591298641078465964189926313473653",
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
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### Allocations

#### allocations-v10

```
{
	"AllocationID": 0,
	"Client": 0,
	"Data": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
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
	"Messages": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"Miner": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Ticket": {
		"VRFProof": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		}
	},
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```


### BlockMessage

#### block-message

```
{
	"Epoch": 0,
	"Messages": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### ChangedActor

#### changed-actor

```
{
	"ActorID": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Address": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Balance": "1024",
	"Code": "",
	"Epoch": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### ChangedClaim

#### changed-claim

```
{
	"Added": false,
	"ClaimID": 0,
	"Client": 0,
	"Data": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"Epoch": 0,
	"Provider": 0,
	"Removed": false,
	"Sector": 0,
	"Size": 0,
	"TermMax": 0,
	"TermMin": 0,
	"TermStart": 0,
	"_id": ""
}
```


### ChangedSector

#### changed-sector

```
{
	"Activation": 0,
	"Added": false,
	"DealIDs": [
		0
	],
	"DealWeight": "1073741824",
	"Epoch": 0,
	"ExpectedDayReward": "1073741824",
	"ExpectedStoragePledge": "340282591298641078465964189926313473653",
	"Expiration": 0,
	"InitialPledge": "1024",
	"Miner": "073366",
	"Removed": false,
	"ReplacedDayReward": "1024",
	"ReplacedSectorAge": 0,
	"SealProof": 0,
	"SealedCID": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"SectorKeyCID": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"SectorNumber": 0,
	"SimpleQAPower": false,
	"VerifiedDealWeight": "340282591298641078465964189926313473653",
	"_id": ""
}
```


### ClaimedPower

#### claimed-power-v0

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### claimed-power-v10

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

#### claimed-power-v11

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

#### claimed-power-v2

```
{
	"Addr": "073366",
	"Detail": {
		"QualityAdjPower": "340282591298641078465964189926313473653",
		"RawBytePower": "1073741824",
		"SealProofType": 0
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### claimed-power-v3

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
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### claimed-power-v4

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

#### claimed-power-v5

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

#### claimed-power-v6

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

#### claimed-power-v7

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
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```

#### claimed-power-v8

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

#### claimed-power-v9

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


### Claims

#### claims-v10

```
{
	"ClaimID": 0,
	"Client": 0,
	"Data": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
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


### DatacapAllowances

#### datacap-allowances-v10

```
{
	"Amount": "340282591298641078465964189926313473653",
	"Epoch": 0,
	"Operator": 0,
	"Owner": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```

#### datacap-allowances-v9

```
{
	"Amount": "1024",
	"Epoch": 0,
	"Operator": 0,
	"Owner": 0,
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### DatacapBalances

#### datacap-balances-v10

```
{
	"Amount": "1073741824",
	"Epoch": 0,
	"Owner": 0,
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
}
```

#### datacap-balances-v9

```
{
	"Amount": "340282591298641078465964189926313473653",
	"Epoch": 0,
	"Owner": 0,
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### DealProposal

#### deal-proposal-full

```
{
	"Client": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"ClientCollateral": "340282591298641078465964189926313473653",
	"ClientID": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"EndEpoch": 0,
	"Epoch": 0,
	"Label": "",
	"PieceCID": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
	"PieceSize": 0,
	"Provider": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"ProviderCollateral": "1073741824",
	"ProviderID": "073366",
	"StartEpoch": 0,
	"StoragePricePerEpoch": "1024",
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
			"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
		],
		"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
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
			"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
		],
		"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	},
	"Detail": {
		"Regular": {
			"ClientCollateral": "340282591298641078465964189926313473653",
			"Clients": 0,
			"Count": 0,
			"PieceSize": "1024",
			"ProviderCollateral": "1073741824",
			"Providers": 0
		},
		"Verified": {
			"ClientCollateral": "340282591298641078465964189926313473653",
			"Clients": 0,
			"Count": 0,
			"PieceSize": "1024",
			"ProviderCollateral": "1073741824",
			"Providers": 0
		}
	}
}
```


### EthHash

#### eth-hash

```
{
	"Cid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
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
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


### EvmInitCode

#### evm-initcode

```
{
	"Epoch": 0,
	"InitCode": "",
	"_id": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q"
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
		"BaseFeeBurn": "340282591298641078465964189926313473653",
		"GasUsed": "1073741824",
		"Message": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
		"MinerPenalty": "1073741824",
		"MinerTip": "340282591298641078465964189926313473653",
		"OverEstimationBurn": "1024",
		"Refund": "1024",
		"TotalCost": "1073741824"
	},
	"IsBlock": false,
	"Msg": {
		"From": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
		"Method": 2,
		"MethodName": "",
		"To": "073366",
		"Value": "1024"
	},
	"MsgRct": {
		"EventsRoot": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq",
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
	"SignedCid": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
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
	"From": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"MethodName": "",
	"To": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"Value": "340282591298641078465964189926313473653",
	"_id": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
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
		"FilReserveDisbursed": "340282591298641078465964189926313473653",
		"FilVested": "1024"
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
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"ClientUnLockCollateralInFuture": [
			"1073741824"
		],
		"ClientUnlockStorageFeeInFuture": [
			"1024"
		],
		"ProviderUnLockCollateralInFuture": [
			"340282591298641078465964189926313473653"
		],
		"TotalClientLockedCollateral": "1073741824",
		"TotalClientStorageFee": "1024",
		"TotalLocked": "1024",
		"TotalProviderLockedCollateral": "340282591298641078465964189926313473653"
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
			"To": "073366",
			"Value": "1073741824"
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
			"To": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
			"Value": "340282591298641078465964189926313473653"
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
	"DealWeight": "1024",
	"Epoch": 0,
	"InitialPledge": "340282591298641078465964189926313473653",
	"Miner": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"QAPower": "1024",
	"SealProof": 0,
	"SectorNumber": 0,
	"VerifiedDealWeight": "1073741824",
	"_id": ""
}
```


### MinerFunds

#### miner-funds

```
{
	"Addr": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	"Detail": {
		"FeeDebt": "1024",
		"InitialPledge": "1073741824",
		"LockedFunds": "340282591298641078465964189926313473653",
		"PledgeRelease": [
			"1024"
		],
		"PreCommitDeposits": "1073741824",
		"VestInFuture": [
			"340282591298641078465964189926313473653"
		]
	},
	"Epoch": 0,
	"Info": {
		"AvailableBalance": "340282591298641078465964189926313473653",
		"Balance": "1073741824",
		"Beneficiary": "073366",
		"BeneficiaryTerm": {
			"Expiration": 0,
			"Quota": "1073741824",
			"UsedQuota": "340282591298641078465964189926313473653"
		},
		"ConsensusFaultElapsed": 0,
		"ControlAddresses": [
			"2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q"
		],
		"FeeDebt": "1024",
		"Multiaddrs": [
			{
				"$binary": {
					"base64": "SGVsbG8=",
					"subType": "00"
				}
			}
		],
		"Owner": "073366",
		"PeerID": {
			"$binary": {
				"base64": "SGVsbG8=",
				"subType": "00"
			}
		},
		"PendingBeneficiaryTerm": {
			"ApprovedByBeneficiary": false,
			"ApprovedByNominee": false,
			"NewBeneficiary": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
			"NewExpiration": 0,
			"NewQuota": "1024"
		},
		"PendingOwnerAddress": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
		"PendingWorkerKey": {
			"EffectiveAt": 0,
			"NewWorker": "3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla"
		},
		"PrecommitSectorCount": 0,
		"SectorSize": 0,
		"State": null,
		"WindowPoStPartitionSectors": 0,
		"WindowPoStProofType": 0,
		"Worker": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq"
	},
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
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
	"DealWeight": "1073741824",
	"Epoch": 0,
	"Expiration": 0,
	"InitialPledge": "1024",
	"Miner": "2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
	"SectorNumber": 0,
	"SimpleQaPower": false,
	"Terminated": false,
	"VerifiedDealWeight": "340282591298641078465964189926313473653",
	"_id": ""
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
				"TotalDealWeight": "1073741824",
				"TotalInitialPledge": "1024",
				"TotalV1InitialPledge": "1073741824",
				"TotalVerifiedDealWeight": "340282591298641078465964189926313473653",
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
		"ExpectedDayReward": "340282591298641078465964189926313473653",
		"InitialConsensusPledge": "1073741824",
		"InitialPledge": "1024",
		"InitialStoragePledge": "340282591298641078465964189926313473653",
		"Mined": "340282591298641078465964189926313473653",
		"ProjectionOfFaultFee": "1073741824",
		"ProjectionOfInitialPledge": "1024"
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
		"Locked": "1024",
		"VestInFuture": [
			"340282591298641078465964189926313473653"
		],
		"Vested": "1073741824"
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
		"Value": "1024"
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"_id": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
}
```


### SectorClaim

#### sector-claim

```
{
	"Client": 0,
	"Data": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
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
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744"
	],
	"_id": 0
}
```


### Tipset

#### tipset

```
{
	"BaseFee": "340282591298641078465964189926313473653",
	"ChildEpoch": 0,
	"Cids": [
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
	],
	"MinTimestamp": 0,
	"Receipts": "bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744",
	"State": "bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw",
	"Weight": "1073741824",
	"_id": 0
}
```


### VerifiedRegistry

#### verifreg

```
{
	"Addr": "1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
	"Detail": {
		"Cap": "1024",
		"Type": ""
	},
	"Epoch": 0,
	"Path": [
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw"
	],
	"_id": "bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq"
}
```


