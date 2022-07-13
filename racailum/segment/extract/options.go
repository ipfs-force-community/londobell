package extract

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
)

// DefaultOptions returns defaults
func DefaultOptions() Options {
	return Options{
		TipSet:        defaultTipSetOptions(),
		StateRegular:  defaultActorStateRegularOptions(),
		EnabelExtract: defaultEnableExtractOptions(),
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
	TipSet        TipSetOptions
	StateRegular  ActorStateRegularOptions
	EnabelExtract EnableExtractOptions
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
		VerifRegTicks:            4,                    // 4h
		MinerSectorSummaryTicks:  24,                   // 24h
		DealProposalSummaryTicks: 12,                   // 12h
		MarketFundsTicks:         24,                   // 24h
		MinerSectorHeathTicks:    1,                    // 1h
		DealProposalDetailTicks:  12,                   // 12h
		ActorBalance:             24,                   // 24h
		PendingTxnsTicks:         1,                    // 1h
	}
}

func dryActorStateRegularOptions() ActorStateRegularOptions {
	return ActorStateRegularOptions{
		Interval:                 1,
		MinerFundsTicks:          1,
		VerifRegTicks:            1,
		MinerSectorSummaryTicks:  1,
		DealProposalSummaryTicks: 1,
		MarketFundsTicks:         1,
		MinerSectorHeathTicks:    1,
		DealProposalDetailTicks:  1,
		ActorBalance:             1,
		PendingTxnsTicks:         1,
	}
}

// ActorStateRegularOptions for actor state extracting
type ActorStateRegularOptions struct {
	Interval                 abi.ChainEpoch
	MinerFundsTicks          int
	VerifRegTicks            int
	MinerSectorSummaryTicks  int
	DealProposalSummaryTicks int
	DealProposalDetailTicks  int
	MarketFundsTicks         int
	MinerSectorHeathTicks    int
	ActorBalance             int
	PendingTxnsTicks         int
}

func IsExtract(tickOption int, ctx *Ctx, curEpoch abi.ChainEpoch) bool {
	if tickOption > 0 && curEpoch%(abi.ChainEpoch(tickOption)*ctx.Opts.StateRegular.Interval) != 0 {
		return false
	}

	return true
}

type EnableExtractOptions struct {
	EnableExtractTrace        bool
	EnableExtractMessage      bool
	EnableExtractTipset       bool
	EnableExtractState        bool
	EnableExtractFilSupply    bool
	EnableExtractActorBalance bool
}

func defaultEnableExtractOptions() EnableExtractOptions {
	return EnableExtractOptions{
		EnableExtractTrace:        true,
		EnableExtractMessage:      true,
		EnableExtractTipset:       true,
		EnableExtractState:        true,
		EnableExtractFilSupply:    true,
		EnableExtractActorBalance: true,
	}
}
