package segment

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

type extractOptions struct {
	Confidence  abi.ChainEpoch
	MinPeriod   abi.ChainEpoch
	MaxBackward abi.ChainEpoch

	TipSetJobLimit int
	StateJobLimit  int

	TipSetPartSizeLimit int

	ExtractOptions extract.Options
}

type persistOptions struct {
	Async            bool
	BatchInsertLimit int
}

// Options for segment
type Options struct {
	Extract extractOptions

	Persist persistOptions

	Aggregate aggregate.Options
}

// DefaultOptions returns a default instance of the Options
func DefaultOptions() Options {
	opt := Options{
		Extract: extractOptions{
			Confidence:  10,
			MinPeriod:   20,
			MaxBackward: 50000,

			TipSetJobLimit: 8,
			StateJobLimit:  32,

			TipSetPartSizeLimit: 16,

			ExtractOptions: extract.DefaultOptions(),
		},

		Persist: persistOptions{
			Async:            true,
			BatchInsertLimit: 4 << 10,
		},
	}

	return opt
}

// OptionFn modifies some field of the options
type OptionFn func(*Options)

// TipSetJobLimit override the TipSetJobLimit by the given non-zero limit
func TipSetJobLimit(limit int) OptionFn {
	return func(opt *Options) {
		if limit > 0 {
			opt.Extract.TipSetJobLimit = limit
		}
	}
}

// StateJobLimit override the StateJobLimit by the given non-zero limit
func StateJobLimit(limit int) OptionFn {
	return func(opt *Options) {
		if limit > 0 {
			opt.Extract.StateJobLimit = limit
		}
	}
}
