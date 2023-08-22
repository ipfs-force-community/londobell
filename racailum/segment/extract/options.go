package extract

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
)

// DefaultOptions returns defaults
func DefaultOptions() Options {
	return Options{
		TipSet:             defaultTipSetOptions(),
		StateRegular:       defaultActorStateRegularOptions(),
		EnabelExtract:      defaultEnableExtractOptions(),
		ZeroHourExtract:    defaultZeroHourExtractOptions(),
		SkipExpensiveEpoch: true,
	}
}

// DryOptions is the options for dry-state run
func DryOptions() Options {
	opt := DefaultOptions()
	opt.StateRegular = dryActorStateRegularOptions()
	return opt
}

// Options for differect extracting jobs
type Options struct {
	TipSet             TipSetOptions
	StateRegular       ActorStateRegularOptions
	EnabelExtract      EnableExtractOptions
	ZeroHourExtract    ZeroHourExtractOptions
	SkipExpensiveEpoch bool
}

func defaultTipSetOptions() TipSetOptions {
	return TipSetOptions{}
}

// TipSetOptions for tipset extracting
type TipSetOptions struct {
}

// ActorStateDiffOptions for actor state extracting
type ActorStateDiffOptions struct {
	Interval abi.ChainEpoch
}

func defaultActorStateRegularOptions() ActorStateRegularOptions {
	return ActorStateRegularOptions{
		Interval:                 builtin.EpochsInHour, // 1h
		MinerFundsTicks:          4,                    // 4h
		VerifiedRegistryTicks:    4,                    // 4h
		MinerSectorSummaryTicks:  24,                   // 24h
		DealProposalSummaryTicks: 12,                   // 12h
		MarketFundsTicks:         24,                   // 24h
		MinerSectorHeathTicks:    1,                    // 1h
		DealProposalDetailTicks:  12,                   // 12h
		ActorBalanceTicks:        24,                   // 24h
		PendingTxnsTicks:         1,                    // 1h
		AllocationsTicks:         4,                    //4h
		ClaimsTicks:              4,                    //4h
		DatacapBalancesTicks:     4,                    //4h
		DatacapAllowancesTicks:   4,                    //4h
		ActorAddressTicks:        1,                    // 1h
	}
}

func dryActorStateRegularOptions() ActorStateRegularOptions {
	return ActorStateRegularOptions{
		Interval:                 1,
		MinerFundsTicks:          1,
		VerifiedRegistryTicks:    1,
		MinerSectorSummaryTicks:  1,
		DealProposalSummaryTicks: 1,
		MarketFundsTicks:         1,
		MinerSectorHeathTicks:    1,
		DealProposalDetailTicks:  1,
		ActorBalanceTicks:        1,
		PendingTxnsTicks:         1,
		AllocationsTicks:         1,
		ClaimsTicks:              1,
		DatacapBalancesTicks:     1,
		DatacapAllowancesTicks:   1,
	}
}

// ActorStateRegularOptions for actor state extracting
type ActorStateRegularOptions struct {
	Interval                 abi.ChainEpoch
	MinerFundsTicks          int
	VerifiedRegistryTicks    int
	MinerSectorSummaryTicks  int
	DealProposalSummaryTicks int
	DealProposalDetailTicks  int
	MarketFundsTicks         int
	MinerSectorHeathTicks    int
	ActorBalanceTicks        int
	PendingTxnsTicks         int
	AllocationsTicks         int
	ClaimsTicks              int
	DatacapBalancesTicks     int
	DatacapAllowancesTicks   int
	ActorAddressTicks        int
}

func IsExtract(tickOption int, ctx *Ctx, curEpoch abi.ChainEpoch) bool {
	if tickOption > 0 && curEpoch%(abi.ChainEpoch(tickOption)*ctx.Opts.StateRegular.Interval) != 0 {
		return false
	}

	return true
}

type EnableExtractOptions struct {
	EnableExtractExecTrace    bool
	EnableExtractMessage      bool
	EnableExtractTipset       bool
	EnableExtractState        bool
	EnableExtractFilSupply    bool
	EnableExtractActorBalance bool
	EnableExtractBlockHeader  bool
	//EnableExtractMessageBlock bool
	EnableExtractBlockMessage     bool
	EnableExtractActorMessage     bool
	EnableExtractEthHash          bool
	EnableExtractEventsRoot       bool
	EnableExtractExplicitMessage  bool
	EnableExtractEvmByteCode      bool
	EnableExtractActorEvent       bool
	EnableExtractMinerSector      bool // 由于需要取未来sectorinfo，故临时库不抽； 变化延迟15分钟能接受吗？
	EnableExtractSectorClaim      bool // 同上
	EnableExtractActorAddress     bool
	EnableExtractChangedActor     bool
	EnableExtractChangedSector    bool
	EnableExtractAllSectors       bool
	EnableExtractChangedClaim     bool
	EnableExtractAllClaims        bool
	EnableExtractNewDealProposal  bool
	EnableExtractChangedDealState bool
	EnableExtractAllDealStates    bool
}

func defaultEnableExtractOptions() EnableExtractOptions {
	return EnableExtractOptions{
		EnableExtractExecTrace:    true,
		EnableExtractMessage:      true,
		EnableExtractTipset:       true,
		EnableExtractState:        true,
		EnableExtractFilSupply:    true,
		EnableExtractActorBalance: true,
		EnableExtractBlockHeader:  true,
		//EnableExtractMessageBlock: true,
		EnableExtractBlockMessage:     true,
		EnableExtractActorMessage:     true,
		EnableExtractEthHash:          true,
		EnableExtractEventsRoot:       true,
		EnableExtractExplicitMessage:  false,
		EnableExtractEvmByteCode:      true,
		EnableExtractActorEvent:       true,
		EnableExtractMinerSector:      true,
		EnableExtractSectorClaim:      true,
		EnableExtractActorAddress:     true,
		EnableExtractChangedActor:     true,
		EnableExtractChangedSector:    true,
		EnableExtractAllSectors:       false,
		EnableExtractChangedClaim:     true,
		EnableExtractAllClaims:        false,
		EnableExtractNewDealProposal:  true,
		EnableExtractChangedDealState: false,
		EnableExtractAllDealStates:    false,
	}
}

type ZeroHourExtractOptions struct {
	ActorBalance        bool
	ClaimedPower        bool
	DealProposalDetail  bool // contain DealProposal
	DealProposalSummary bool
	FilSupply           bool
	MarketFunds         bool
	MinerFunds          bool
	MinerSectorHealth   bool
	MinerSectorSummary  bool // contain MinerDealSector
	MiningProfitability bool
	MultisigBalance     bool
	PendingTxns         bool
	VerifiedRegistry    bool
	Allocation          bool
	Claims              bool
	DatacapAllowances   bool
	DatacapBalances     bool
	ActorAddress        bool
}

func defaultZeroHourExtractOptions() ZeroHourExtractOptions {
	return ZeroHourExtractOptions{
		ActorBalance:        true,
		ClaimedPower:        true,
		DealProposalDetail:  true,
		DealProposalSummary: true,
		FilSupply:           true,
		MarketFunds:         true,
		MinerFunds:          true,
		MinerSectorHealth:   true,
		MinerSectorSummary:  true,
		MiningProfitability: true,
		MultisigBalance:     true,
		PendingTxns:         true,
		VerifiedRegistry:    true,
		Allocation:          true,
		Claims:              true,
		DatacapAllowances:   true,
		DatacapBalances:     true,
	}
}
