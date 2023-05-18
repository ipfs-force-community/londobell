package aggregators

import (
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
)

var (
	addressAggregator                       []byte
	aggPreNetfeeAggregator                  []byte
	aggProNetfeeAggregator                  []byte
	blockAggregator                         []byte
	finalHeightAggregator                   []byte
	minerBlockrewardAggregator              []byte
	minersInfoAggregator                    []byte
	minersMinedAggregator                   []byte
	multisigMessageAggregator               []byte
	punishmentAggregator                    []byte
	wincountZlAggregator                    []byte
	wincountForMinerAggregator              []byte
	tracesAggregator                        []byte
	childEpochAggregator                    []byte
	minersBlockrewardAggregator             []byte
	burnMonitorAggregator                   []byte
	latestTipSetAggregator                  []byte
	totalBlockCountAggregator               []byte
	actorStateAggregator                    []byte
	tipsetAggregator                        []byte
	minerInfoAggregator                     []byte
	balanceAggregator                       []byte
	minersForOwnerAggregator                []byte
	messagesForActorAggregator              []byte
	transferMessagesAggregator              []byte
	timeOfTraceAggregator                   []byte
	createTimeAggregator                    []byte
	createMessageAggregator                 []byte
	gasCostForSectorAggregator              []byte
	transferMessageForLargeAmountAggregator []byte
	dealsAggregator                         []byte
	dealByIDAggregator                      []byte
	dealsByAddrAggregator                   []byte
	detailForDealAggregator                 []byte
	blockHeaderAggregator                   []byte
	traceForMessageAggregator               []byte
	batchTraceForMessageAggregator          []byte
	childTransfersForMessageAggregator      []byte
	allOwnersAggregator                     []byte
	parentTipSetAggregator                  []byte
	blockHeaderByCidAggregator              []byte
	blockMessagesByMethodNameAggregator     []byte
	actorMessagesByMethodNameAggregator     []byte
	blockHeadersByMinerAggregator           []byte
	minedByMinerRangeAggregator             []byte
	//allMethodsAggregator                                 []byte
	blocksForMessageAggregator                           []byte
	countAndMethodNameOfMessagesForBlockHeaderAggregator []byte
	messagesForBlockAggregator                           []byte
	countOfMessagesForBlockHeaderByMethodNameAggregator  []byte
	blockHeaderMessagesByMethodNameAggregator            []byte
	richListAggregator                                   []byte
	allActorsForBlockMessageAggregator                   []byte
	//transferCountForActorAggregator                      []byte
	countOfTransfersForActor2Aggregator   []byte
	countOfLargeAmountTransfersAggregator []byte
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
	wincountForMinerAggregator = monitor.GetWincountForMinerAggregator()
	tracesAggregator = monitor.GetTracesAggregator()
	childEpochAggregator = monitor.GetChildEpochAggregator()
	minersBlockrewardAggregator = monitor.GetMinersBlockRewardAggregator()
	burnMonitorAggregator = monitor.GetBurnMonitorAggregator()
	latestTipSetAggregator = monitor.GetLatestTipSetAggregator()
	totalBlockCountAggregator = monitor.GetTotalBlockCountAggregator()
	actorStateAggregator = monitor.GetActorStateAggregator()
	tipsetAggregator = monitor.GetTipSetAggregator()
	minerInfoAggregator = monitor.GetMinerInfoAggregator()
	balanceAggregator = monitor.GetBalanceAggregator()
	minersForOwnerAggregator = monitor.GetMinersForOwnerAggregator()
	messagesForActorAggregator = monitor.GetMessagesForActorAggregator()
	transferMessagesAggregator = monitor.GetTransferMessagesAggregator()
	timeOfTraceAggregator = monitor.GetTimeOfTraceAggregator()
	createTimeAggregator = monitor.GetCreateTimeAggregator()
	createMessageAggregator = monitor.GetCreateMessageAggregator()
	gasCostForSectorAggregator = monitor.GetGasCostForSectorAggregator()
	transferMessageForLargeAmountAggregator = monitor.GetTransferMessageForLargeAmountAggregator()
	dealsAggregator = monitor.GetDealsAggregator()
	dealByIDAggregator = monitor.GetDealByIDAggregator()
	detailForDealAggregator = monitor.GetDetailForDealAggregator()
	blockHeaderAggregator = monitor.GetBlockHeaderAggregator()
	traceForMessageAggregator = monitor.GetTraceForMessageAggregator()
	batchTraceForMessageAggregator = monitor.GetBatchTraceForMessageAggregator()
	childTransfersForMessageAggregator = monitor.GetChildTransfersForMessageAggregator()
	allOwnersAggregator = monitor.GetAllOwnersAggregator()
	parentTipSetAggregator = monitor.GetParentTipSetAggregator()
	blockHeaderByCidAggregator = monitor.GetBlockHeaderByCidAggregator()
	blockMessagesByMethodNameAggregator = monitor.GetBlockMessagesByMethodNameAggregator()
	actorMessagesByMethodNameAggregator = monitor.GetActorMessagesByMethodNameAggregator()
	blockHeadersByMinerAggregator = monitor.GetBlockHeadersByMinerAggregator()
	minedByMinerRangeAggregator = monitor.GetMinedByMinerRangeAggregator()
	dealsByAddrAggregator = monitor.GetDealsByAddrAggregator()
	//allMethodsAggregator = monitor.GetAllMethodsAggregator()
	blocksForMessageAggregator = monitor.GetBlocksForMessageAggregator()
	countAndMethodNameOfMessagesForBlockHeaderAggregator = monitor.GetCountAndMethodNameOfMessagesForBlockHeaderAggregator()
	messagesForBlockAggregator = monitor.GetMessagesForBlockAggregator()
	countOfMessagesForBlockHeaderByMethodNameAggregator = monitor.GetCountOfMessagesForBlockHeaderByMethodNameAggregator()
	blockHeaderMessagesByMethodNameAggregator = monitor.GetBlockHeaderMessagesByMethodNameAggregator()
	richListAggregator = monitor.GetRichListAggregator()
	allActorsForBlockMessageAggregator = monitor.GetAllActorsForBlockMessageAggregator()
	//transferCountForActorAggregator = monitor.GetTransferCountForActorAggregator()
	countOfTransfersForActor2Aggregator = monitor.GetCountOfTransfersForActor2Aggregator()
	countOfLargeAmountTransfersAggregator = monitor.GetCountOfLargeAmountTransfersAggregator()
}
