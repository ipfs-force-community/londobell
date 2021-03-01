package extract

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
)

// DefaultOptions returns defaults
func DefaultOptions() Options {
	return Options{
		TipSet:       defaultTipSetOptions(),
		StateDiff:    defaultActorStateDiffOptions(),
		StateRegular: defaultActorStateRegularOptions(),
	}
}

// Options for differect extracting jobs
type Options struct {
	TipSet       TipSetOptions
	StateDiff    ActorStateDiffOptions
	StateRegular ActorStateRegularOptions
}

func defaultTipSetOptions() TipSetOptions {
	return TipSetOptions{}
}

// TipSetOptions for tipset extracting
type TipSetOptions struct {
}

func defaultActorStateDiffOptions() ActorStateDiffOptions {
	return ActorStateDiffOptions{
		Interval: 0, // just disable state diff extraction here
	}
}

// ActorStateDiffOptions for actor state extracting
type ActorStateDiffOptions struct {
	Interval abi.ChainEpoch
}

func defaultActorStateRegularOptions() ActorStateRegularOptions {
	return ActorStateRegularOptions{
		Interval:        builtin.EpochsInHour, // 1h
		MinerFundsTicks: 4,                    // 4h
		VerifRegTicks:   4,                    // 4h
	}
}

// ActorStateRegularOptions for actor state extracting
type ActorStateRegularOptions struct {
	Interval        abi.ChainEpoch
	MinerFundsTicks int
	VerifRegTicks   int
}
