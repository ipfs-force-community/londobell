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
	messagesForActorAggregator              []byte //nolint
	transferMessagesAggregator              []byte //nolint
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
	actorMessagesByMethodNameAggregator     []byte //nolint
	blockHeadersByMinerAggregator           []byte //nolint
	minedByMinerRangeAggregator             []byte
	//allMethodsAggregator                                 []byte
	blocksForMessageAggregator                           []byte
	countAndMethodNameOfMessagesForBlockHeaderAggregator []byte
	messagesForBlockAggregator                           []byte
	countOfMessagesForBlockHeaderByMethodNameAggregator  []byte
	blockHeaderMessagesByMethodNameAggregator            []byte
	richListAggregator                                   []byte
	//allActorsForBlockMessageAggregator                   []byte
	//transferCountForActorAggregator                      []byte
	countOfTransfersForActor2Aggregator      []byte //nolint
	countOfLargeAmountTransfersAggregator    []byte //nolint
	transferMsgsForActorAggregator           []byte
	childCallsForMessageAggregator           []byte
	eventsByActorAggregator                  []byte
	eventsForMessageAggregator               []byte
	eventsForEpochRangeAggregator            []byte
	transferBlockRewardForActorAggregator    []byte
	transferBurnForActorAggregator           []byte
	transferSendAndReceiveForActorAggregator []byte
	transferSendForActorAggregator           []byte
	transferReceiveForActorAggregator        []byte
	tipsetsListAggregator                    []byte

	actorMessageNoSkip []byte
	//countActorMessage  []byte //nolint
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
	minersBlockrewardAggregator = monitor.GetMinersBlockrewardAggregator()
	burnMonitorAggregator = monitor.GetBurnMonitorAggregator()
	latestTipSetAggregator = monitor.GetLatestTipsetAggregator()
	totalBlockCountAggregator = monitor.GetTotalBlockCountAggregator()
	actorStateAggregator = monitor.GetActorStateAggregator()
	tipsetAggregator = monitor.GetTipsetAggregator()
	minerInfoAggregator = monitor.GetMinerInfoAggregator()
	balanceAggregator = monitor.GetBalanceAggregator()
	minersForOwnerAggregator = monitor.GetMinersForOwnerAggregator()
	messagesForActorAggregator = monitor.GetMessagesForActorAggregator()
	transferMessagesAggregator = monitor.GetTransferMessagesAggregator()
	timeOfTraceAggregator = monitor.GetTimeOfTraceAggregator()
	createTimeAggregator = monitor.GetCreatetimeAggregator()
	createMessageAggregator = monitor.GetCreateMessageAggregator()
	gasCostForSectorAggregator = monitor.GetGascostForSectorAggregator()
	transferMessageForLargeAmountAggregator = monitor.GetTransferMessageForLargeAmountAggregator()
	dealsAggregator = monitor.GetDealsAggregator()
	dealByIDAggregator = monitor.GetDealByIdAggregator()
	detailForDealAggregator = monitor.GetDetailForDealAggregator()
	blockHeaderAggregator = monitor.GetBlockheaderAggregator()
	traceForMessageAggregator = monitor.GetTraceForMessageAggregator()
	batchTraceForMessageAggregator = monitor.GetBatchTraceForMessageAggregator()
	childTransfersForMessageAggregator = monitor.GetChildTransfersForMessageAggregator()
	allOwnersAggregator = monitor.GetAllOwnersAggregator()
	parentTipSetAggregator = monitor.GetParentTipsetAggregator()
	blockHeaderByCidAggregator = monitor.GetBlockheaderByCidAggregator()
	blockMessagesByMethodNameAggregator = monitor.GetBlockmessagesByMethodnameAggregator()
	actorMessagesByMethodNameAggregator = monitor.GetActormessagesByMethodnameAggregator()
	blockHeadersByMinerAggregator = monitor.GetBlockheadersByMinerAggregator()
	minedByMinerRangeAggregator = monitor.GetMinedByMinerRangeAggregator()
	dealsByAddrAggregator = monitor.GetDealsByAddrAggregator()
	//allMethodsAggregator = monitor.GetAllMethodsAggregator()
	blocksForMessageAggregator = monitor.GetBlocksForMessageAggregator()
	countAndMethodNameOfMessagesForBlockHeaderAggregator = monitor.GetCountAndMethodnamesOfMessagesForBlockheaderAggregator()
	messagesForBlockAggregator = monitor.GetMessagesForBlockAggregator()
	countOfMessagesForBlockHeaderByMethodNameAggregator = monitor.GetCountOfMessagesForBlockheaderByMethodnameAggregator()
	blockHeaderMessagesByMethodNameAggregator = monitor.GetBlockheadermessagesByMethodnameAggregator()
	richListAggregator = monitor.GetRichlistAggregator()
	//allActorsForBlockMessageAggregator = monitor.GetAllActorsForBlockMessageAggregator()
	//transferCountForActorAggregator = monitor.GetTransferCountForActorAggregator()
	countOfTransfersForActor2Aggregator = monitor.GetCountOfTransfersForActor2Aggregator()
	countOfLargeAmountTransfersAggregator = monitor.GetCountOfLargeamountTransfersAggregator()
	transferMsgsForActorAggregator = monitor.GetTransfermsgsForActorAggregator()
	childCallsForMessageAggregator = monitor.GetChildCallsForMessageAggregator()
	eventsByActorAggregator = monitor.GetEventsByActorAggregator()
	eventsForMessageAggregator = monitor.GetEventsForMessageAggregator()
	eventsForEpochRangeAggregator = monitor.GetEventsForEpochrangeAggregator()
	transferBlockRewardForActorAggregator = monitor.GetTransferBlockrewardForActorAggregator()
	transferBurnForActorAggregator = monitor.GetTransferBurnForActorAggregator()
	transferSendAndReceiveForActorAggregator = monitor.GetTransferSendAndReceiveForActorAggregator()
	transferSendForActorAggregator = monitor.GetTransferSendForActorAggregator()
	transferReceiveForActorAggregator = monitor.GetTransferReceiveForActorAggregator()
	tipsetsListAggregator = monitor.GetTipsetsListAggregator()

	actorMessageNoSkip = monitor.GetMessagesForActorNoSkipAggregator()
}
