package aggregators

import (
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
)

var (
	addressAggregator           []byte
	aggPreNetfeeAggregator      []byte
	aggProNetfeeAggregator      []byte
	blockAggregator             []byte
	finalHeightAggregator       []byte
	minerBlockrewardAggregator  []byte
	minersInfoAggregator        []byte
	minersMinedAggregator       []byte
	multisigMessageAggregator   []byte
	punishmentAggregator        []byte
	wincountZlAggregator        []byte
	tracesAggregator            []byte
	childEpochAggregator        []byte
	minersBlockrewardAggregator []byte
	burnMonitorAggregator       []byte
	latestTipSetAggregator      []byte
	totalBlockCountAggregator   []byte
	actorStateAggregator        []byte
	tipsetAggregator            []byte
	//claimedPowerForMinerAggregator []byte
	minerInfoAggregator                                  []byte
	balanceAggregator                                    []byte
	minersForOwnerAggregator                             []byte
	messagesForActorAggregator                           []byte
	transferMessagesAggregator                           []byte
	timeOfTraceAggregator                                []byte
	createTimeAggregator                                 []byte
	gasCostForSectorAggregator                           []byte
	transferMessageForLargeAmountAggregator              []byte
	dealsAggregator                                      []byte
	detailForDealAggregator                              []byte
	blockHeaderAggregator                                []byte
	traceForMessageAggregator                            []byte
	batchTraceForMessageAggregator                       []byte
	childTransfersForMessageAggregator                   []byte
	allOwnersAggregator                                  []byte
	parentTipSetAggregator                               []byte
	blockHeaderByCidAggregator                           []byte
	blockMessagesByMethodNameAggregator                  []byte
	actorMessagesByMethodNameAggregator                  []byte
	blockHeadersByMinerAggregator                        []byte
	dealsByAddrAggregator                                []byte
	allMethodsAggregator                                 []byte
	allMethodsForActorAggregator                         []byte
	blocksForMessageAggregator                           []byte
	countAndMethodNameOfMessagesForBlockHeaderAggregator []byte
	messagesForBlockAggregator                           []byte
	countOfMessagesForBlockHeaderByMethodNameAggregator  []byte
	blockHeaderMessagesByMethodNameAggregator            []byte
	richListAggregator                                   []byte
	dealByIDAggregator                                   []byte

	BlockMsgsMap        map[abi.ChainEpoch]int64            // record totalCount for formal db, latestEpoch: blockMessagesCount todo: make
	AllMethods          map[abi.ChainEpoch][]string         // latestEpoch: allMethods
	BlockMsgByMethodMap map[abi.ChainEpoch]map[string]int64 // LatestEpoch: {methodName,totalCount} 热库递增缓存？

	// actor 消息列表
	AllActors map[abi.ChainEpoch]map[string]string // 记录一段时间内发/接块消息的actors latestEpoch:{ID: robust}

	ActorMsgsMap        map[abi.ChainEpoch]map[string]int64            // latestEpoch:{actor(key是ID 也包含robust的消息): totalCount}  actor有多少条消息数
	AllActorMethods     map[abi.ChainEpoch]map[string][]string         // latestEpoch:{actor(ID 包含robust的): allMethods}  actor有各种类型消息方法
	ActorMsgByMethodMap map[abi.ChainEpoch]map[string]map[string]int64 // latestEpoch:{actor: {method: count}} 					actor根据消息方法筛选有多少条数
	Mutex               sync.Mutex
)

type Methodlist struct {
	methodName string
	Count      int64
}

type Addresslist struct {
	ActorID    string
	RobustAddr string
}

type RobustCountlist struct {
	RobustAddr string
	Count      int64
}
type ActorIDsMap struct {
	ActorIDRobusts map[string]string
}

func InitAggregators() {
	// todo: 定期重新读取，无感知变化 or 每次变化重启
	addressAggregator = monitor.GetAddressAggregator()
	aggPreNetfeeAggregator = monitor.GetAggPreNetfeeAggregator()
	aggProNetfeeAggregator = monitor.GetAggProNetfeeAggregator()
	blockAggregator = monitor.GetBlockAggregator()
	finalHeightAggregator = monitor.GetFinalHeightAggregator()
	minerBlockrewardAggregator = monitor.GetMinerBlockrewardAggregator()
	minersInfoAggregator = monitor.GetMinersInfoAggregator()
	minersMinedAggregator = monitor.GetMinersMinedAggregator()
	multisigMessageAggregator = monitor.GetMultisigMessageAggregator()
	punishmentAggregator = monitor.GetPunishmentAggregator()
	wincountZlAggregator = monitor.GetWincountZlAggregator()
	tracesAggregator = monitor.GetTracesAggregator()
	batchTraceForMessageAggregator = monitor.GetBatchTraceForMessageAggregator()
	childEpochAggregator = monitor.GetChildEpochAggregator()
	minersBlockrewardAggregator = monitor.GetMinersBlockRewardAggregator()
	burnMonitorAggregator = monitor.GetBurnMonitorAggregator()
	latestTipSetAggregator = monitor.GetLatestTipSetAggregator()
	totalBlockCountAggregator = monitor.GetTotalBlockCountAggregator()
	actorStateAggregator = monitor.GetActorStateAggregator()
	tipsetAggregator = monitor.GetTipSetAggregator()
	//claimedPowerForMinerAggregator = monitor.GetClaimedPowerForMinerAggregator()
	minerInfoAggregator = monitor.GetMinerInfoAggregator()
	balanceAggregator = monitor.GetBalanceAggregator()
	minersForOwnerAggregator = monitor.GetMinersForOwnerAggregator()
	messagesForActorAggregator = monitor.GetMessagesForActorAggregator()
	transferMessagesAggregator = monitor.GetTransferMessagesAggregator()
	timeOfTraceAggregator = monitor.GetTimeOfTraceAggregator()
	createTimeAggregator = monitor.GetCreateTimeAggregator()
	gasCostForSectorAggregator = monitor.GetGasCostForSectorAggregator()
	transferMessageForLargeAmountAggregator = monitor.GetTransferMessageForLargeAmountAggregator()
	dealsAggregator = monitor.GetDealsAggregator()
	detailForDealAggregator = monitor.GetDetailForDealAggregator()
	blockHeaderAggregator = monitor.GetBlockHeaderAggregator()
	traceForMessageAggregator = monitor.GetTraceForMessageAggregator()
	childTransfersForMessageAggregator = monitor.GetChildTransfersForMessage()
	allOwnersAggregator = monitor.GetAllOwnerAggregator()
	parentTipSetAggregator = monitor.GetParentTipSetAggregator()
	blockHeaderByCidAggregator = monitor.GetBlockHeaderByCidAggregator()
	blockMessagesByMethodNameAggregator = monitor.GetBlockMessagesByMethodNameAggregator()
	actorMessagesByMethodNameAggregator = monitor.GetActorMessagesByMethodNameAggregator()
	blockHeadersByMinerAggregator = monitor.GetBlockHeadersByMinerAggregator()
	dealsByAddrAggregator = monitor.GetDealsByAddrAggregator()
	allMethodsAggregator = monitor.GetAllMethodsAggregator()
	allMethodsForActorAggregator = monitor.GetAllMethodsForActorAggregator()
	blocksForMessageAggregator = monitor.GetBlocksForMessageAggregator()
	countAndMethodNameOfMessagesForBlockHeaderAggregator = monitor.GetCountAndMethodNameOfMessagesForBlockHeaderAggregator()
	messagesForBlockAggregator = monitor.GetMessagesForBlockAggregator()
	countOfMessagesForBlockHeaderByMethodNameAggregator = monitor.GetCountOfMessagesForBlockHeaderByMethodNameAggregator()
	blockHeaderMessagesByMethodNameAggregator = monitor.GetBlockHeaderMessagesByMethodNameAggregator()
	richListAggregator = monitor.GetRichListAggregator()
	dealByIDAggregator = monitor.GetDealByIDAggregator()
}
