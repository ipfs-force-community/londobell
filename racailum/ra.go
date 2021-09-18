package racailum

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/grafana"
	"github.com/dtynn/londobell/racailum/segment"
	"github.com/dtynn/londobell/racailum/segment/aggregate"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

var log = logging.Logger("racailum")

// SegmentConfig contains info about a segment
type SegmentConfig struct {
	DSN     string
	ReadDSN string
	Name    string
}

// DefaultConfig returns the default config
func DefaultConfig() Config {
	return Config{
		Grafana:          grafana.DefaultOptions(),
		Aggregate:        aggregate.DefaultOptions(),
		Segment:          segment.DefaultOptions(),
		EnableGasTracing: false,
		EnableGrafana:    true,
	}
}

// Config of RaCailum
type Config struct {
	Grafana          grafana.Options
	Aggregate        aggregate.Options
	Segment          segment.Options
	EnableGasTracing bool
	EnableGrafana    bool
}

// New returns an instance of *RaCailum
func New(ctx context.Context, cfg Config, sub common.HeadNotifier, cs common.ChainStore, stm common.StateManager, segmgr *segment.Manager) (*RaCailum, error) {
	vm.EnableGasTracing = cfg.EnableGasTracing

	activeSegName, has, err := segmgr.LoadActive()
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, fmt.Errorf("no active segment")
	}

	activeSeg, err := segment.New(ctx, activeSegName, cfg.Segment, cfg.Aggregate, segmgr, cs, stm)
	if err != nil {
		return nil, err
	}

	// TODO: fix grafana with multi seg
	gr, err := grafana.New(ctx, cfg.Grafana, []*segment.Segment{activeSeg})
	if err != nil {
		return nil, fmt.Errorf("construct garfana: %w", err)
	}

	if cfg.EnableGrafana {
		go func() {
			if err := gr.Run(ctx); err != nil {
				log.Errorf("grafana: %s", err)
			}
		}()
	}

	log.Infow("ra sets sail", "gas-tracing", cfg.EnableGasTracing, "active-seg", activeSegName)
	ra := &RaCailum{
		cfg:       cfg,
		sub:       sub,
		activeSeg: activeSeg,
	}

	ra.components.cs = cs
	ra.components.stm = stm

	ra.gr = gr

	return ra, nil
}

// RaCailum manages the segments
type RaCailum struct {
	cfg Config

	sub common.HeadNotifier

	components struct {
		cs     common.ChainStore
		stm    common.StateManager
		optfns []segment.OptionFn
	}

	gr        *grafana.Grafana
	activeSeg *segment.Segment
}

// Extract is used to extract from given tipset manually
func (r *RaCailum) Extract(ctx context.Context, ts *types.TipSet) error {
	return r.activeSeg.Extract(ctx, ts)
}

// Aggregate is used to trigger aggregations from given tipset boundaries manually
func (r *RaCailum) Aggregate(ctx context.Context, lo, hi *types.TipSet) error {
	loEpoch := lo.Height()
	tss, err := segment.ExtractLinkedTipSets(r.components.cs, hi, &loEpoch)
	if err != nil {
		return fmt.Errorf("load tipsets: %w", err)
	}
	// TODO: fix me
	//return r.activeSeg.Aggregate(ctx, tss)

	_ = tss
	return nil
}

// DryState runs a dry extraction from given ts
func (r *RaCailum) DryState(ctx context.Context, ts *common.LinkedTipSet) ([]*extract.Res, error) {
	return r.activeSeg.DryExtract(ctx, ts)
}

// Grafana returns the instance of *Grafana
func (r *RaCailum) Grafana() *grafana.Grafana {
	return r.gr
}
