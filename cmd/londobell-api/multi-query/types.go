package multiquery

type CountUtil struct {
	Start int64
	End   int64
	Count int64

	Cols Collections

	Tmp    bool
	Formal bool

	InnerStates []InnerState
}

type InnerState struct {
	Start int64
	End   int64
	Count int64
}

type aggUtil struct {
	startEpoch int64
	endEpoch   int64
	skip       int64
	limit      int64

	cols Collections

	InnerStates []InnerState
}

//type ByMethodNameUtil struct {
//	StartEpoch abi.ChainEpoch // 整个库查询的开始
//	EndEpoch   abi.ChainEpoch // 已缓存的高度+1
//
//	latestHeightForBlockMsg abi.ChainEpoch
//
//	latestEpochForAllMethods          abi.ChainEpoch // 从StartEpoch开始。 上次刷新缓存的高度+1，下次缓存的开始 todo: 更名为nextStartEpochForAllMethods
//	latestEpochForBlockMsgByMethodMap abi.ChainEpoch
//
//	latestEpochForAllActorMethods     abi.ChainEpoch
//	latestEpochForActorMsgByMethodMap abi.ChainEpoch
//
//	latestEpochForAllActors    abi.ChainEpoch
//	latestEpochForActorMsgsMap abi.ChainEpoch
//
//	latestEpochForAllActorsForTransfers abi.ChainEpoch
//	latestEpochForAllActorTransfersMap  abi.ChainEpoch
//
//	//latestEpochForAllMinedMiners    abi.ChainEpoch
//	latestEpochForMinedMinerMsgsMap abi.ChainEpoch
//
//	latestEpochForTransfersLargeAmount abi.ChainEpoch
//
//	blockFilter    interface{}
//	blockMsgsCount int64
//
//	allMethods          []string
//	blockMsgByMethodMap map[string]int64
//
//	allActorMethods     map[string][]string
//	actorMsgByMethodMap map[string]map[string]int64
//
//	allActors    map[string]addresses
//	actorMsgsMap map[string]int64
//
//	allActorsForTransfers map[string]addresses
//	allActorTransfersMap  map[string]int64
//
//	// 出块
//	//allMinedMiners     map[string]addresses
//	minedMinersMsgsMap map[string]int64
//
//	transfersLargeAmountCount  int64
//	transfersLargeAmountFilter interface{}
//
//	traceCol       *mongo.Collection
//	messageCol     *mongo.Collection
//	BlockHeaderCol *mongo.Collection
//
//	col *mongo.Collection
//}
//
//type addresses struct {
//	robust    string
//	delegated string
//}
