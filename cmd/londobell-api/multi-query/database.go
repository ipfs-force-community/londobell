package multiquery

import (
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

var EmptyStateCid cid.Cid

// Adapts string as a mapping key
type DBKey string

func (k DBKey) Key() string {
	return string(k)
}

type DataBaseState struct {
	Formal bool `bson:"Formal"`
	Tmp    bool `bson:"Tmp"`

	StartEpoch abi.ChainEpoch `bson:"StartEpoch"` // 整个库查询的开始
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`   // finalHeight+1, 右开   每两周刷一次

	//RefreshStartEpoch abi.ChainEpoch // 每次刷新结束的EndEpoch赋值

	//NextEpochForBlockMsgsCount        abi.ChainEpoch // 对于tmp，总是等于StartEpoch
	//NextEpochForBlockMsgsByMethodName abi.ChainEpoch
	//NextEpochForActorMsgsByMethodName abi.ChainEpoch
	//NextEpochForActorMsgsCount        abi.ChainEpoch
	//NextEpochForActorTransfersCount   abi.ChainEpoch
	//NextEpochForMinedMsgs             abi.ChainEpoch
	//NextEpochForTransfersLargeAmount  abi.ChainEpoch

	//InnerDBStates []InnerDBState // 内部范围分区

	BlockMsgsStates             []BlockMsgsState             `bson:"BlockMsgsStates"`
	BlockMsgsByMethodNameStates []BlockMsgsByMethodNameState `bson:"BlockMsgsByMethodNameStates"`
	ActorMsgsByMethodNameStates []ActorMsgsByMethodNameState `bson:"ActorMsgsByMethodNameStates"`
	ActorMsgsCountStates        []ActorMsgsCountState        `bson:"ActorMsgsCountStates"`
	ActorTransfersCountStates   []ActorTransfersCountState   `bson:"ActorTransfersCountStates"`
	MinedMsgsStates             []MinedMsgsState             `bson:"MinedMsgsStates"`
	TransfersLargeAmountStates  []TransfersLargeAmountState  `bson:"TransfersLargeAmountStates"`
}

//type InnerKey struct {
//	StartEpoch abi.ChainEpoch // 整个库查询的开始
//	EndEpoch   abi.ChainEpoch // finalHeight+1, 右开
//}

type BlockMsgsState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// 块消息
	BlockMsgsCount int64 `bson:"BlockMsgsCount"`
}

type BlockMsgsByMethodNameState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// 方法名筛选块消息
	BlockMsgsByMethodNameMap map[string]int64 `bson:"BlockMsgsByMethodNameMap"`
}

type ActorMsgsByMethodNameState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// 方法名筛选actor消息
	ActorMsgsByMethodNameMap map[string]map[string]int64 `bson:"ActorMsgsByMethodNameMap"` // methodName: actorID
}

type ActorMsgsCountState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// actor消息
	ActorMsgsCountMap map[string]int64 `bson:"ActorMsgsCountMap"`
}

type ActorTransfersCountState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// actor转账消息
	ActorTransfersCountMap map[string]int64 `bson:"ActorTransfersCountMap"`
}

type MinedMsgsState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// 出块列表
	MinedMsgsMap map[string]int64 `bson:"MinedMsgsMap"`
}

type TransfersLargeAmountState struct {
	StartEpoch abi.ChainEpoch `bson:"StartEpoch"`
	EndEpoch   abi.ChainEpoch `bson:"EndEpoch"`
	// 大额转账列表
	TransfersLargeAmountCount int64 `bson:"TransfersLargeAmountCount"`
}

type DBCollections struct {
	DBCollectionsMap map[string]Collections
	Lock             sync.Mutex
}

func DefaultBlockMsgsStates(startEpoch abi.ChainEpoch) []BlockMsgsState {
	blockMsgsStates := make([]BlockMsgsState, 0)
	blockMsgsStates = append(blockMsgsStates, BlockMsgsState{StartEpoch: startEpoch, EndEpoch: startEpoch, BlockMsgsCount: 0})

	return blockMsgsStates
}

func DefaultBlockMsgsByMethodNameStates(startEpoch abi.ChainEpoch) []BlockMsgsByMethodNameState {
	blockMsgsByMethodNameStates := make([]BlockMsgsByMethodNameState, 0)
	blockMsgsByMethodNameStates = append(blockMsgsByMethodNameStates, BlockMsgsByMethodNameState{StartEpoch: startEpoch, EndEpoch: startEpoch, BlockMsgsByMethodNameMap: make(map[string]int64)})

	return blockMsgsByMethodNameStates
}

func DefaultActorMsgsCountStates(startEpoch abi.ChainEpoch) []ActorMsgsCountState {
	actorMsgsCountStates := make([]ActorMsgsCountState, 0)
	actorMsgsCountStates = append(actorMsgsCountStates, ActorMsgsCountState{StartEpoch: startEpoch, EndEpoch: startEpoch, ActorMsgsCountMap: make(map[string]int64)})

	return actorMsgsCountStates
}

func DefaultActorMsgsByMethodNameStates(startEpoch abi.ChainEpoch) []ActorMsgsByMethodNameState {
	actorMsgsByMethodNameStates := make([]ActorMsgsByMethodNameState, 0)
	actorMsgsByMethodNameStates = append(actorMsgsByMethodNameStates, ActorMsgsByMethodNameState{StartEpoch: startEpoch, EndEpoch: startEpoch, ActorMsgsByMethodNameMap: make(map[string]map[string]int64)})

	return actorMsgsByMethodNameStates
}

func DefaultActorTransfersCountStates(startEpoch abi.ChainEpoch) []ActorTransfersCountState {
	actorTransfersCountStates := make([]ActorTransfersCountState, 0)
	actorTransfersCountStates = append(actorTransfersCountStates, ActorTransfersCountState{StartEpoch: startEpoch, EndEpoch: startEpoch, ActorTransfersCountMap: make(map[string]int64)})

	return actorTransfersCountStates
}

func DefaultMinedMsgsStates(startEpoch abi.ChainEpoch) []MinedMsgsState {
	minedMsgsStates := make([]MinedMsgsState, 0)
	minedMsgsStates = append(minedMsgsStates, MinedMsgsState{StartEpoch: startEpoch, EndEpoch: startEpoch, MinedMsgsMap: make(map[string]int64)})

	return minedMsgsStates
}

func DefaultTransfersLargeAmountStates(startEpoch abi.ChainEpoch) []TransfersLargeAmountState {
	transfersLargeAmountStates := make([]TransfersLargeAmountState, 0)
	transfersLargeAmountStates = append(transfersLargeAmountStates, TransfersLargeAmountState{StartEpoch: startEpoch, EndEpoch: startEpoch, TransfersLargeAmountCount: 0})

	return transfersLargeAmountStates
}

func DefaultDataBaseState(formal, tmp bool, start, end abi.ChainEpoch) *DataBaseState {
	return &DataBaseState{
		Formal:                      formal,
		Tmp:                         tmp,
		StartEpoch:                  start,
		EndEpoch:                    end,
		BlockMsgsStates:             DefaultBlockMsgsStates(start),
		BlockMsgsByMethodNameStates: DefaultBlockMsgsByMethodNameStates(start),
		ActorMsgsByMethodNameStates: DefaultActorMsgsByMethodNameStates(start),
		ActorMsgsCountStates:        DefaultActorMsgsCountStates(start),
		ActorTransfersCountStates:   DefaultActorTransfersCountStates(start),
		MinedMsgsStates:             DefaultMinedMsgsStates(start),
		TransfersLargeAmountStates:  DefaultTransfersLargeAmountStates(start),
	}
}

// 更新formal，新增cold   异步程序更新cold state,   中间会有一段时间数据断层？
