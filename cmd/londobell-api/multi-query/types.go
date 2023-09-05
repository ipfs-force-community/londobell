package multiquery

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

type Ptype int
type CtxKey string

const (
	BlockStates Ptype = iota
	BlockMethodStates
	BlockHeaderMethodStates
	ActorStates
	ActorMethodStates
	ActorTransferStates
	ActorEventStates
	MinedStates
	LargeAmountTransferStates
	DealState
	DealActorStates
	TipSetStates
	AllStates
)

// use for context
var TableKey CtxKey = "tableName"

type CountUtil struct {
	Start int64
	End   int64

	Cols common.Collections

	DType smodel.DType

	BlockStates []smodel.SegmentState

	// 暂时不对其他state分段
	BlockMethodStates         []smodel.SegmentState
	BlockHeaderMethodStates   int64
	ActorStates               int64
	ActorMethodStates         int64
	ActorTransferStates       int64
	ActorEventStates          int64
	MinedStates               int64
	LargeAmountTransferStates int64

	DealState       int64 // todo： 暂时不分段，测试下
	DealActorStates int64
	TipSetStates    int64

	SectorState int64 // new sector count
}

type segmentUtil struct {
	start int64
	end   int64
	skip  int64
	limit int64

	Cols common.Collections

	BlockStates []smodel.SegmentState

	// 暂时不对其他state分段
	BlockMethodStates         []smodel.SegmentState
	BlockHeaderMethodStates   int64
	ActorStates               int64
	ActorMethodStates         int64
	ActorTransferStates       int64
	ActorEventStates          int64
	MinedStates               int64
	LargeAmountTransferStates int64
	DealState                 int64
	DealActorStates           int64
	TipSetStates              int64
}

type aggUtil struct {
	start int64
	end   int64
	skip  int64
	limit int64

	cols common.Collections

	count int64
}

type ByMethodNameUtil struct {
	StartEpoch abi.ChainEpoch // 整个库查询的开始
	EndEpoch   abi.ChainEpoch // 已缓存的高度+1

	latestHeightForBlockMsg abi.ChainEpoch

	latestEpochForAllMethods          abi.ChainEpoch // 从StartEpoch开始。 上次刷新缓存的高度+1，下次缓存的开始 todo: 更名为nextStartEpochForAllMethods
	latestEpochForBlockMsgByMethodMap abi.ChainEpoch

	latestEpochForAllActorMethods     abi.ChainEpoch
	latestEpochForActorMsgByMethodMap abi.ChainEpoch

	latestEpochForAllActors    abi.ChainEpoch
	latestEpochForActorMsgsMap abi.ChainEpoch

	latestEpochForAllActorsForTransfers abi.ChainEpoch
	latestEpochForAllActorTransfersMap  abi.ChainEpoch

	//latestEpochForAllMinedMiners    abi.ChainEpoch
	latestEpochForMinedMinerMsgsMap abi.ChainEpoch

	latestEpochForTransfersLargeAmount abi.ChainEpoch

	blockFilter    interface{}
	blockMsgsCount int64

	allMethods          []string
	blockMsgByMethodMap map[string]int64

	allActorMethods     map[string][]string
	actorMsgByMethodMap map[string]map[string]int64

	allActors    map[string]addresses
	actorMsgsMap map[string]int64

	allActorsForTransfers map[string]addresses
	allActorTransfersMap  map[string]int64

	// 出块
	//allMinedMiners     map[string]addresses
	minedMinersMsgsMap map[string]int64

	transfersLargeAmountCount  int64
	transfersLargeAmountFilter interface{}

	traceCol       *mongo.Collection
	messageCol     *mongo.Collection
	BlockHeaderCol *mongo.Collection

	col *mongo.Collection
}

type addresses struct {
	robust    string
	delegated string
}
