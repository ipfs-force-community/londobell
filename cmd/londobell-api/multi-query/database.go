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

//type DataBaseState struct {
//	Formal bool
//	Tmp    bool
//
//	StartEpoch abi.ChainEpoch // 整个库查询的开始
//	EndEpoch   abi.ChainEpoch // finalHeight+1, 右开
//
//	// 块消息
//	NextEpochForBlockMsgsCount abi.ChainEpoch // 对于tmp，总是等于StartEpoch
//	BlockMsgsCount             int64
//
//	// 方法名筛选块消息
//	NextEpochForBlockMsgsByMethodName abi.ChainEpoch
//	BlockMsgsByMethodNameMap          map[string]int64
//
//	// 方法名筛选actor消息
//	NextEpochForActorMsgsByMethodName abi.ChainEpoch
//	ActorMsgsByMethodNameMap          map[string]map[string]int64 // methodName: actorID
//
//	// actor消息
//	NextEpochForActorMsgsCount abi.ChainEpoch
//	ActorMsgsCountMap          map[string]int64
//
//	// actor转账消息
//	NextEpochForActorTransfersCount abi.ChainEpoch
//	ActorTransfersCountMap          map[string]int64
//
//	// 出块列表
//	NextEpochForMinedMsgs abi.ChainEpoch
//	MinedMsgsMap          map[string]int64
//
//	// 大额转账列表
//	NextEpochForTransfersLargeAmount abi.ChainEpoch
//	TransfersLargeAmountCount        int64
//}

type DataBaseState struct {
	Formal bool
	Tmp    bool

	StartEpoch abi.ChainEpoch // 整个库查询的开始
	EndEpoch   abi.ChainEpoch // finalHeight+1, 右开

	// 块消息
	BlockMsgsCount int64
	// 方法名筛选块消息
	BlockMsgsByMethodNameMap map[string]int64
	// 方法名筛选actor消息
	ActorMsgsByMethodNameMap map[string]map[string]int64 // methodName: actorID
	// actor消息
	ActorMsgsCountMap map[string]int64
	// actor转账消息
	ActorTransfersCountMap map[string]int64
	// 出块列表
	MinedMsgsMap map[string]int64
	// 大额转账列表
	TransfersLargeAmountCount int64

	//// todo: delete
	//NextEpochForBlockMsgsCount        abi.ChainEpoch // 对于tmp，总是等于StartEpoch
	//NextEpochForBlockMsgsByMethodName abi.ChainEpoch
	//NextEpochForActorMsgsByMethodName abi.ChainEpoch
	//NextEpochForActorMsgsCount        abi.ChainEpoch
	//NextEpochForActorTransfersCount   abi.ChainEpoch
	//NextEpochForMinedMsgs             abi.ChainEpoch
	//NextEpochForTransfersLargeAmount  abi.ChainEpoch
}

type DBCollections struct {
	DBCollectionsMap map[string]Collections
	Lock             sync.Mutex
}

func DefaultDataBaseState(formal, tmp bool, start, end abi.ChainEpoch) *DataBaseState {
	return &DataBaseState{
		Formal:                    formal,
		Tmp:                       tmp,
		StartEpoch:                start,
		EndEpoch:                  end,
		BlockMsgsCount:            0,
		BlockMsgsByMethodNameMap:  make(map[string]int64),
		ActorMsgsByMethodNameMap:  make(map[string]map[string]int64),
		ActorMsgsCountMap:         make(map[string]int64),
		ActorTransfersCountMap:    make(map[string]int64),
		MinedMsgsMap:              make(map[string]int64),
		TransfersLargeAmountCount: 0,
	}
}

// 更新formal，新增cold   异步程序更新cold state,   中间会有一段时间数据断层？
