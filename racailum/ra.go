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

// Config of RaCailum
type Config struct {
	Grafana          grafana.Options
	Aggregate        aggregate.Options
	EnableGasTracing bool
	EnableGrafana    bool
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

		var segDocDB common.DocumentDB
		var segDict common.ChainDict

		{
			segdb := cli.Database(scfg.Name)
			segDocDB, err = mgoutil.NewMgoDocDB(ctx, cli, segdb)
			if err != nil {
				return nil, fmt.Errorf("construct doc db for %s: %w", scfg.Name, err)
			}

			segDict, err = mdict.NewDict(segdb)
			if err != nil {
				return nil, fmt.Errorf("construct dict for %s: %w", scfg.Name, err)
			}
		}

		segDocDBRead := segDocDB
		if scfg.ReadDSN != "" {
			readcli, err := mgoutil.Connect(ctx, scfg.ReadDSN)
			if err != nil {
				return nil, fmt.Errorf("connect to segment database for read %s: %w", scfg.ReadDSN, err)
			}

			segdbRead := readcli.Database(scfg.Name)
			segDocDBRead, err = mgoutil.NewMgoDocDB(ctx, readcli, segdbRead)
			if err != nil {
				return nil, fmt.Errorf("construct doc db for read for %s: %w", scfg.Name, err)
			}
		}

		seg, err := segment.New(scfg.Name, cfg.Aggregate, segDocDB, segDocDBRead, metamgr, cs, segDict, stm, optfns...)
		if err != nil {
			return nil, fmt.Errorf("construct segment %s: %w", scfg.Name, err)
		}

		segments = append(segments, seg)
	}

	gr, err := grafana.New(ctx, cfg.Grafana, segments)
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

	ra.gr = gr

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

	gr        *grafana.Grafana
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

// DryState runs a dry extraction from given ts
func (r *RaCailum) DryState(ctx context.Context, ts *common.LinkedTipSet) ([]*extract.Res, error) {
	return r.segments[r.activeseg].DryExtract(ctx, ts)
}

// Grafana returns the instance of *Grafana
func (r *RaCailum) Grafana() *grafana.Grafana {
	return r.gr
}
