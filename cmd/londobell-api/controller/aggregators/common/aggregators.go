package common

import (
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
)

var (
	AddressAggregator                       []byte
	AggPreNetfeeAggregator                  []byte
	AggProNetfeeAggregator                  []byte
	BlockAggregator                         []byte
	finalHeightAggregator                   []byte
	MinerBlockrewardAggregator              []byte
	MinersInfoAggregator                    []byte
	MinersMinedAggregator                   []byte
	MultisigMessageAggregator               []byte
	PunishmentAggregator                    []byte
	WincountZlAggregator                    []byte
	WincountForMinerAggregator              []byte
	TracesAggregator                        []byte
	ChildEpochAggregator                    []byte
	MinersBlockrewardAggregator             []byte
	BurnMonitorAggregator                   []byte
	LatestTipSetAggregator                  []byte
	TotalBlockCountAggregator               []byte
	ActorStateAggregator                    []byte
	TipsetAggregator                        []byte
	MinerInfoAggregator                     []byte
	BalanceAggregator                       []byte
	MinersForOwnerAggregator                []byte
	MessagesForActorAggregator              []byte
	transferMessagesAggregator              []byte
	TimeOfTraceAggregator                   []byte
	CreateTimeAggregator                    []byte
	CreateMessageAggregator                 []byte
	GasCostForSectorAggregator              []byte
	TransferMessageForLargeAmountAggregator []byte
	DealsAggregator                         []byte
	DealByIDAggregator                      []byte
	DealsByAddrAggregator                   []byte
	DetailForDealAggregator                 []byte
	BlockHeaderAggregator                   []byte
	TraceForMessageAggregator               []byte
	BatchTraceForMessageAggregator          []byte
	ChildTransfersForMessageAggregator      []byte
	AllOwnersAggregator                     []byte
	ParentTipSetAggregator                  []byte
	BlockHeaderByCidAggregator              []byte
	BlockMessagesByMethodNameAggregator     []byte
	ActorMessagesByMethodNameAggregator     []byte
	BlockHeadersByMinerAggregator           []byte
	MinedByMinerRangeAggregator             []byte
	//allMethodsAggregator                                 []byte
	BlocksForMessageAggregator                           []byte
	CountAndMethodNameOfMessagesForBlockHeaderAggregator []byte
	MessagesForBlockAggregator                           []byte
	CountOfMessagesForBlockHeaderByMethodNameAggregator  []byte
	BlockHeaderMessagesByMethodNameAggregator            []byte
	RichListAggregator                                   []byte
	//allActorsForBlockMessageAggregator                   []byte
	//transferCountForActorAggregator                      []byte
	countOfTransfersForActor2Aggregator            []byte
	countOfLargeAmountTransfersAggregator          []byte
	TransferMsgsForActorAggregator                 []byte
	ChildCallsForMessageAggregator                 []byte
	EventsByActorAggregator                        []byte
	EventsForMessageAggregator                     []byte
	EventsForEpochRangeAggregator                  []byte
	TransferBlockRewardForActorAggregator          []byte
	TransferBurnForActorAggregator                 []byte
	TransferSendAndReceiveForActorAggregator       []byte
	TransferSendForActorAggregator                 []byte
	TransferReceiveForActorAggregator              []byte
	TipsetsListAggregator                          []byte
	ActorMessageNoSkip                             []byte
	TransferMsgsFroActorNoSkipAggregator           []byte
	TransferTypeForActorNoSkipAggregator           []byte
	TransferSendAndReceiveForActorNoSkipAggregator []byte
	TimeOfCreateAggregator                         []byte

	MinerPeriodBlockrewardsAggregator            []byte
	MinerPeriodWincountsAggregator               []byte
	MinerPeriodGascostsAggregator                []byte
	MinerPeriodGascostsForPublishdealsAggregator []byte
	MinerPeriodPunishmentsAggregator             []byte
	MinerPeriodSectorsDiffAggregator             []byte
	MinerPeriodPledgeDiffAggregator              []byte
	MinerPeriodSectorExpirationsAggregator       []byte
	MinerQAPowerHistoryAggregator                []byte
	MinerSectorhealthHistoryAggregator           []byte
	MinerPledgeHistoryAggregtor                  []byte
	MinerSectorRangeAggregator                   []byte
	MinerPeriodSectorExpirationAggregator        []byte
	MinerSectorDetailAggregator                  []byte
	AccountPeriodTransferAggregator              []byte
	AccountPeriodGasCostAggregator               []byte
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
	AddressAggregator = monitor.GetAddressAggregator()
	AggPreNetfeeAggregator = monitor.GetAggPreNetfeeAggregator()
	AggProNetfeeAggregator = monitor.GetAggProNetfeeAggregator()
	BlockAggregator = monitor.GetBlockAggregator()
	finalHeightAggregator = monitor.GetFinalHeightAggregator()
	MinerBlockrewardAggregator = monitor.GetMinerBlockrewardAggregator()
	MinersInfoAggregator = monitor.GetMinersInfoAggregator()
	MinersMinedAggregator = monitor.GetMinersMinedAggregator()
	MultisigMessageAggregator = monitor.GetMultisigMessageAggregator()
	PunishmentAggregator = monitor.GetPunishmentAggregator()
	WincountZlAggregator = monitor.GetWincountZlAggregator()
	WincountForMinerAggregator = monitor.GetWincountForMinerAggregator()
	TracesAggregator = monitor.GetTracesAggregator()
	ChildEpochAggregator = monitor.GetChildEpochAggregator()
	MinersBlockrewardAggregator = monitor.GetMinersBlockRewardAggregator()
	BurnMonitorAggregator = monitor.GetBurnMonitorAggregator()
	LatestTipSetAggregator = monitor.GetLatestTipSetAggregator()
	TotalBlockCountAggregator = monitor.GetTotalBlockCountAggregator()
	ActorStateAggregator = monitor.GetActorStateAggregator()
	TipsetAggregator = monitor.GetTipSetAggregator()
	MinerInfoAggregator = monitor.GetMinerInfoAggregator()
	BalanceAggregator = monitor.GetBalanceAggregator()
	MinersForOwnerAggregator = monitor.GetMinersForOwnerAggregator()
	MessagesForActorAggregator = monitor.GetMessagesForActorAggregator()
	transferMessagesAggregator = monitor.GetTransferMessagesAggregator()
	TimeOfTraceAggregator = monitor.GetTimeOfTraceAggregator()
	CreateTimeAggregator = monitor.GetCreateTimeAggregator()
	CreateMessageAggregator = monitor.GetCreateMessageAggregator()
	GasCostForSectorAggregator = monitor.GetGasCostForSectorAggregator()
	TransferMessageForLargeAmountAggregator = monitor.GetTransferMessageForLargeAmountAggregator()
	DealsAggregator = monitor.GetDealsAggregator()
	DealByIDAggregator = monitor.GetDealByIDAggregator()
	DetailForDealAggregator = monitor.GetDetailForDealAggregator()
	BlockHeaderAggregator = monitor.GetBlockHeaderAggregator()
	TraceForMessageAggregator = monitor.GetTraceForMessageAggregator()
	BatchTraceForMessageAggregator = monitor.GetBatchTraceForMessageAggregator()
	ChildTransfersForMessageAggregator = monitor.GetChildTransfersForMessageAggregator()
	AllOwnersAggregator = monitor.GetAllOwnersAggregator()
	ParentTipSetAggregator = monitor.GetParentTipSetAggregator()
	BlockHeaderByCidAggregator = monitor.GetBlockHeaderByCidAggregator()
	BlockMessagesByMethodNameAggregator = monitor.GetBlockMessagesByMethodNameAggregator()
	ActorMessagesByMethodNameAggregator = monitor.GetActorMessagesByMethodNameAggregator()
	BlockHeadersByMinerAggregator = monitor.GetBlockHeadersByMinerAggregator()
	MinedByMinerRangeAggregator = monitor.GetMinedByMinerRangeAggregator()
	DealsByAddrAggregator = monitor.GetDealsByAddrAggregator()
	//allMethodsAggregator = monitor.GetAllMethodsAggregator()
	BlocksForMessageAggregator = monitor.GetBlocksForMessageAggregator()
	CountAndMethodNameOfMessagesForBlockHeaderAggregator = monitor.GetCountAndMethodNameOfMessagesForBlockHeaderAggregator()
	MessagesForBlockAggregator = monitor.GetMessagesForBlockAggregator()
	CountOfMessagesForBlockHeaderByMethodNameAggregator = monitor.GetCountOfMessagesForBlockHeaderByMethodNameAggregator()
	BlockHeaderMessagesByMethodNameAggregator = monitor.GetBlockHeaderMessagesByMethodNameAggregator()
	RichListAggregator = monitor.GetRichListAggregator()
	//allActorsForBlockMessageAggregator = monitor.GetAllActorsForBlockMessageAggregator()
	//transferCountForActorAggregator = monitor.GetTransferCountForActorAggregator()
	countOfTransfersForActor2Aggregator = monitor.GetCountOfTransfersForActor2Aggregator()
	countOfLargeAmountTransfersAggregator = monitor.GetCountOfLargeAmountTransfersAggregator()
	TransferMsgsForActorAggregator = monitor.GetTransferMsgsForActorAggregator()
	ChildCallsForMessageAggregator = monitor.GetChildCallsForMessageAggregator()
	EventsByActorAggregator = monitor.GetEventsByActorAggregator()
	EventsForMessageAggregator = monitor.GetEventsForMessageAggregator()
	EventsForEpochRangeAggregator = monitor.GetEventsForEpochRangeAggregator()
	TransferBlockRewardForActorAggregator = monitor.GetTransferBlockRewardForActorAggregator()
	TransferBurnForActorAggregator = monitor.GetTransferBurnForActorAggregator()
	TransferSendAndReceiveForActorAggregator = monitor.GetTransferSendAndReceiveForActorAggregator()
	TransferSendForActorAggregator = monitor.GetTransferSendForActorAggregator()
	TransferReceiveForActorAggregator = monitor.GetTransferReceiveForActorAggregator()
	TipsetsListAggregator = monitor.GetTipsetsListAggregator()
	ActorMessageNoSkip = monitor.GetMessagesForActorNoSkip()
	TransferMsgsFroActorNoSkipAggregator = monitor.GetTransferMsgsFroActorNoSkipAggregator()
	TransferTypeForActorNoSkipAggregator = monitor.GetTransferTypeForActorNoSkipAggregator()
	TransferSendAndReceiveForActorNoSkipAggregator = monitor.GetTransferSendAndReceiveForActorNoSkipAggregator()
	TimeOfCreateAggregator = monitor.GetTimeOfCreateAggregator()

	MinerPeriodBlockrewardsAggregator = monitor.GetMinerPeriodBlockrewardsAggregator()
	MinerPeriodWincountsAggregator = monitor.GetMinerPeriodWincountsAggregator()
	MinerPeriodGascostsAggregator = monitor.GetMinerPeriodGascostsAggregator()
	MinerPeriodGascostsForPublishdealsAggregator = monitor.GetMinerPeriodGascostsForPublishdealsAggregator()
	MinerPeriodPunishmentsAggregator = monitor.GetMinerPeriodPunishmentsAggregator()
	MinerPeriodSectorsDiffAggregator = monitor.GetMinerPeriodSectorsDiffAggregator()
	MinerPeriodPledgeDiffAggregator = monitor.GetMinerPeriodPledgeDiffAggregator()
	MinerPeriodSectorExpirationsAggregator = monitor.GetMinerPeriodSectorExpirationsAggregator()
	MinerQAPowerHistoryAggregator = monitor.GetMinerQAPowerHistoryAggregator()
	MinerSectorhealthHistoryAggregator = monitor.GetMinerSectorhealthHistoryAggregator()
	MinerPledgeHistoryAggregtor = monitor.GetMinerPledgeHistoryAggregator()
	MinerSectorRangeAggregator = monitor.GetMinerSectorRangeAggregator()
	MinerPeriodSectorExpirationAggregator = monitor.GetMinerPeriodSectorExpirationAggregator()
	MinerSectorDetailAggregator = monitor.GetMinerSectorDetailAggregator()
	AccountPeriodTransferAggregator = monitor.GetAccountPeriodTransferAggregator()
	AccountPeriodGasCostAggregator = monitor.GetAccountPeriodGasCostAggregator()
}
