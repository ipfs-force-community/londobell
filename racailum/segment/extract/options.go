package extract

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"

	"github.com/ipfs-force-community/londobell/common"
)

// DefaultOptions returns defaults
func DefaultOptions() Options {
	return Options{
		TipSet:             defaultTipSetOptions(),
		StateRegular:       defaultActorStateRegularOptions(),
		EnabelExtract:      defaultEnableExtractOptions(),
		ZeroHourExtract:    defaultZeroHourExtractOptions(),
		SkipExpensiveEpoch: true,
		OnlyExtractState:   false,
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
	OnlyExtractState   bool
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

func IsExtract(tickOption int, interval, curEpoch abi.ChainEpoch) bool {
	if tickOption > 0 && curEpoch%(abi.ChainEpoch(tickOption)*interval) != 0 {
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
	EnableExtractBlockMessage         bool
	EnableExtractActorMessage         bool
	EnableExtractEthHash              bool
	EnableExtractEventsRoot           bool
	EnableExtractExplicitMessage      bool
	EnableExtractEvmByteCode          bool
	EnableExtractActorEvent           bool
	EnableExtractMinerSector          bool // 由于需要取未来sectorinfo，故临时库不抽； 变化延迟15分钟能接受吗？
	EnableExtractSectorClaim          bool // 同上
	EnableExtractActorAddress         bool
	EnableExtractChangedActor         bool
	EnableExtractMinerNewSectorNumber bool
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
		EnableExtractBlockMessage: true,

		EnableExtractActorMessage:         true,
		EnableExtractEthHash:              true,
		EnableExtractEventsRoot:           true,
		EnableExtractExplicitMessage:      false,
		EnableExtractEvmByteCode:          true,
		EnableExtractActorEvent:           true,
		EnableExtractMinerSector:          true,
		EnableExtractSectorClaim:          true,
		EnableExtractActorAddress:         true,
		EnableExtractChangedActor:         true,
		EnableExtractMinerNewSectorNumber: true,
	}
}

func SkipExtractExecTrace(opts Options) bool {
	return opts.OnlyExtractState || !opts.EnabelExtract.EnableExtractExecTrace && !opts.EnabelExtract.EnableExtractMessage && !opts.EnabelExtract.EnableExtractActorMessage && !opts.EnabelExtract.EnableExtractEthHash && !opts.EnabelExtract.EnableExtractEventsRoot &&
		!opts.EnabelExtract.EnableExtractExplicitMessage && !opts.EnabelExtract.EnableExtractEvmByteCode && !opts.EnabelExtract.EnableExtractActorEvent && !opts.EnabelExtract.EnableExtractMinerSector && !opts.EnabelExtract.EnableExtractSectorClaim &&
		!opts.EnabelExtract.EnableExtractMinerNewSectorNumber
}

func EnableExtractEthHash(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractEthHash
}

func EnableExtractMessage(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractMessage
}

func EnableExtractExecTrace(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractExecTrace
}

func EnableExtractEvent(opts Options) bool {
	return !opts.OnlyExtractState && (opts.EnabelExtract.EnableExtractEventsRoot || opts.EnabelExtract.EnableExtractActorEvent)
}

func EnableExtractEventsRoot(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractEventsRoot
}

func EnableExtractActorEvent(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractActorEvent
}

func EnableExtractActorMessage(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractActorMessage
}

func EnableExtractExplicitMessage(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractExplicitMessage
}

func EnableExtractEvmByteCode(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractEvmByteCode
}

func EnableExtractSector(tmp bool, opts Options, av actors.Version) bool {
	return !tmp && !opts.OnlyExtractState && (opts.EnabelExtract.EnableExtractMinerSector || EnableExtractSectorClaim(tmp, opts, av) || opts.EnabelExtract.EnableExtractMinerNewSectorNumber)
}

func EnableExtractMinerSector(tmp bool, opts Options) bool {
	return !tmp && !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractMinerSector
}

func EnableExtractSectorClaim(tmp bool, opts Options, av actors.Version) bool {
	return !tmp && !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractSectorClaim && av > actors.Version8
}

func EnableExtractMinerNewSectorNumber(tmp bool, opts Options) bool {
	return !tmp && !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractMinerNewSectorNumber
}

func EnableExtractTipset(tmp bool, opts Options) bool {
	return !tmp && !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractTipset
}

func EnableExtractBlockHeader(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractBlockHeader
}

func EnableExtractActorBalance(tmp bool, opts Options, height abi.ChainEpoch) bool {
	return !tmp && !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractActorBalance && (common.IsZeroHour(opts.ZeroHourExtract.ActorBalance, height) || IsExtract(opts.StateRegular.ActorBalanceTicks, opts.StateRegular.Interval, height))
}

func ForRegular(opts ActorStateRegularOptions, height abi.ChainEpoch) bool {
	return opts.Interval > 0 && height%opts.Interval == 0
}

func SkipExtractActorHead(ctx *Ctx, ts *common.LinkedTipSet, tmp bool) bool {
	if tmp {
		return true
	}

	var extractEvenNullTipSet bool
	for h := ts.Parent.Height() + 1; h <= ts.Height(); h++ {
		if ForRegular(ctx.Opts.StateRegular, h) || common.IsZeroHour(true, h) {
			extractEvenNullTipSet = true
			break
		}
	}

	return (!ctx.Opts.OnlyExtractState && !ctx.Opts.EnabelExtract.EnableExtractState && !ctx.Opts.EnabelExtract.EnableExtractFilSupply) || !extractEvenNullTipSet
}

func EnableExtractState(opts Options) bool {
	return opts.OnlyExtractState || opts.EnabelExtract.EnableExtractState
}

func EnableExtractFilSupply(opts Options, height abi.ChainEpoch) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractFilSupply && (ForRegular(opts.StateRegular, height) || common.IsZeroHour(opts.ZeroHourExtract.FilSupply, height))
}

func EnableExtractBlockMessage(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractBlockMessage
}

func SkipExtractActorAddress(tmp bool, opts Options, height abi.ChainEpoch) bool {
	return tmp || opts.OnlyExtractState || !opts.EnabelExtract.EnableExtractActorAddress || !common.IsZeroHour(opts.ZeroHourExtract.ActorAddress, height) && !IsExtract(opts.StateRegular.ActorAddressTicks, opts.StateRegular.Interval, height)
}

func EnableExtractChangedActor(opts Options) bool {
	return !opts.OnlyExtractState && opts.EnabelExtract.EnableExtractChangedActor
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
		ActorAddress:        true,
	}
}
