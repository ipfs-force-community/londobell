package racailum

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mgoutil"
	"github.com/dtynn/londobell/lib/mgoutil/mdict"
	"github.com/dtynn/londobell/racailum/segment"
	"github.com/dtynn/londobell/racailum/segment/aggregate"
)

var log = logging.Logger("racailum")

// SegmentConfig contains info about a segment
type SegmentConfig struct {
	DSN  string
	Name string
}

// Config of RaCailum
type Config struct {
	Aggregate        aggregate.Options
	EnableGasTracing bool
	Segments         []SegmentConfig
}

// New returns an instance of *RaCailum
func New(ctx context.Context, cfg Config, sub common.HeadNotifier, metamgr common.MetaManager, cs common.ChainStore, stm common.StateManager, optfns ...segment.OptionFn) (*RaCailum, error) {
	vm.EnableGasTracing = cfg.EnableGasTracing

	if len(cfg.Segments) == 0 {
		return nil, fmt.Errorf("no segment config provided")
	}

	segments := make([]*segment.Segment, 0, len(cfg.Segments))
	for i := range cfg.Segments {
		scfg := cfg.Segments[i]
		cli, err := mgoutil.Connect(ctx, scfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("connect to segment database %s: %w", scfg.DSN, err)
		}

		segdb := cli.Database(scfg.Name)
		segDocDB, err := mgoutil.NewMgoDocDB(ctx, cli, segdb)
		segDict, err := mdict.NewDict(segdb)

		seg, err := segment.New(scfg.Name, cfg.Aggregate, segDocDB, metamgr, cs, segDict, stm, optfns...)
		if err != nil {
			return nil, fmt.Errorf("construct segment %s: %w", scfg.Name, err)
		}

		segments = append(segments, seg)
	}

	log.Infow("ra sets sail", "gas-tracing", cfg.EnableGasTracing, "segments", len(segments))
	ra := &RaCailum{
		cfg:       cfg,
		sub:       sub,
		segments:  segments,
		activeseg: len(segments) - 1,
	}

	ra.components.metamgr = metamgr
	ra.components.cs = cs
	ra.components.stm = stm
	ra.components.optfns = optfns

	return ra, nil
}

// RaCailum manages the segments
type RaCailum struct {
	cfg Config

	sub common.HeadNotifier

	components struct {
		metamgr common.MetaManager
		cs      common.ChainStore
		stm     common.StateManager
		optfns  []segment.OptionFn
	}

	segments  []*segment.Segment
	activeseg int
}

// Extract is used to extract from given tipset manually
func (r *RaCailum) Extract(ctx context.Context, ts *types.TipSet) error {
	return r.segments[r.activeseg].Extract(ctx, ts)
}

// Aggregate is used to trigger aggregations from given tipset boundaries manually
func (r *RaCailum) Aggregate(ctx context.Context, lo, hi *types.TipSet) error {
	loEpoch := lo.Height()
	tss, err := segment.ExtractLinkedTipSets(r.components.cs, hi, &loEpoch)
	if err != nil {
		return fmt.Errorf("load tipsets: %w", err)
	}

	return r.segments[r.activeseg].Aggregate(ctx, tss)
}
