package racailum

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum/grafana"
	"github.com/ipfs-force-community/londobell/racailum/segment"
	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/tracing"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	lconfig "github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

var log = logging.Logger("racailum")

const DefaultHTTPListenAddr = ":15002"
const DefaultRPCListenAddr = "/ip4/127.0.0.1/tcp/12345"

// SegmentConfig contains info about a segment
type SegmentConfig struct {
	DSN     string
	ReadDSN string
	Name    string
}

// DefaultConfig returns the default config
func DefaultConfig() Config {
	return Config{
		HTTP:           DefaultHTTPOptions(),
		Grafana:        grafana.DefaultOptions(),
		Aggregate:      aggregate.DefaultOptions(),
		Segment:        segment.DefaultOptions(),
		Metrics:        metrics.DefaultOptions(),
		Tracing:        tracing.DefaultOptions(),
		EnableTracing:  true,
		EnableGrafana:  true,
		EnableDebug:    true,
		OutdatedGap:    240,
		TriggerSpan:    240, // 2h
		TempDBCapacity: 240,
	}
}

type HTTPOptions struct {
	RPCListen  string
	Listen     string
	StableWait lconfig.Duration
}

func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		RPCListen:  DefaultRPCListenAddr,
		Listen:     DefaultHTTPListenAddr,
		StableWait: lconfig.Duration(5 * time.Second),
	}
}

// Config of RaCailum
type Config struct {
	HTTP           HTTPOptions
	Grafana        grafana.Options
	Aggregate      aggregate.Options
	Segment        segment.Options
	Metrics        metrics.Options
	Tracing        tracing.Options
	EnableTracing  bool
	EnableGrafana  bool
	EnableDebug    bool
	OutdatedGap    int64
	TriggerSpan    uint
	TempDBCapacity uint
}

// New returns an instance of *RaCailum
func New(ctx context.Context, cfg Config, sub common.HeadNotifier, cs common.ChainStore, stm common.StateManager, segmgr *segment.Manager, shutdownCh dtypes.ShutdownChan) (*RaCailum, error) {
	vm.EnableDetailedTracing = cfg.EnableTracing

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

	// gr, err := grafana.New(ctx, cfg.Grafana, []*segment.Segment{activeSeg})
	// if err != nil {
	//     return nil, fmt.Errorf("construct garfana: %w", err)
	// }

	// if cfg.EnableGrafana {
	//     go func() {
	//         if err := gr.Run(ctx); err != nil {
	//             log.Errorf("grafana: %s", err)
	//         }
	//     }()
	// }
	log.Infow("ra sets sail", "tracing", cfg.EnableTracing, "active-seg", activeSegName)
	ra := &RaCailum{
		cfg:        cfg,
		sub:        sub,
		activeSeg:  activeSeg,
		shutdownCh: shutdownCh,
	}

	ra.components.cs = cs
	ra.components.stm = stm
	// ra.gr = gr

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

	// gr        *grafana.Grafana
	activeSeg *segment.Segment

	shutdownCh dtypes.ShutdownChan
}

// Extract is used to extract from given tipset manually
func (r *RaCailum) Extract(ctx context.Context, ts *types.TipSet) error {
	return r.activeSeg.Extract(ctx, ts)
}

// OfflineExtract is used to extract from given tipset key (hi, from)
func (r *RaCailum) WalkExtract(ctx context.Context, from types.TipSetKey, hi abi.ChainEpoch) error {
	ts, err := r.components.cs.LoadTipSet(ctx, from)
	if err != nil {
		return fmt.Errorf("load ts failed, is it exist in this db? : %w", err)
	}
	tipsets, err := segment.ExtractLinkedTipSets(r.components.cs, ts, &hi, false)
	if err != nil {
		return fmt.Errorf("extract linked tipsets failed: %w", err)
	}

	for i := 1; i < len(tipsets); i++ {
		tipsets[i].Parent = tipsets[i-1].TipSet
	}

	tipsets = tipsets[1:]

	if err := r.activeSeg.ExtractTipSets(ctx, tipsets, false); err != nil {
		return err
	}

	if err := r.activeSeg.Aggregate(ctx, tipsets); err != nil {
		return err
	}
	return nil
}

// Aggregate is used to trigger aggregations from given tipset boundaries manually
func (r *RaCailum) Aggregate(ctx context.Context, lo, hi *types.TipSet) error {
	loEpoch := lo.Height()
	tss, err := segment.ExtractLinkedTipSets(r.components.cs, hi, &loEpoch, false)
	if err != nil {
		return fmt.Errorf("load tipsets: %w", err)
	}

	return r.activeSeg.Aggregate(ctx, tss)
}

// DryState runs a dry extraction from given ts
func (r *RaCailum) DryState(ctx context.Context, ts *common.LinkedTipSet, allowNilChild bool) ([]*extract.Res, error) {
	return r.activeSeg.DryExtract(ctx, ts, allowNilChild)
}

func (r *RaCailum) Run(ctx context.Context, doneCh <-chan struct{}, tsCh <-chan types.TipSetKey) {
HEAD_LOOP:
	for {
		select {
		case <-doneCh:
			log.Info("quit head-change loop")
			return

		case tsk, ok := <-tsCh:
			if !ok {
				log.Warn("tsk chan closed")
				return
			}
			lstart := time.Now()
			ts, err := r.components.cs.LoadTipSet(ctx, tsk)
			stats.Record(ctx, metrics.LoadTipSetDuration.M(metrics.SinceInMilliseconds(lstart)))
			if err != nil {
				log.Errorf("failed to load tipset %s: %s", tsk, err)
				continue HEAD_LOOP
			}

			log.Infow("incoming tipset", "tsk", tsk, "height", ts.Height())
			estart := time.Now()
			if err := r.Extract(ctx, ts); err != nil {
				log.Errorf("failed to persist tipset: %s", err)
				stats.Record(ctx, metrics.ExtractError.M(1))
			} else {
				stats.Record(ctx, metrics.ExtractError.M(0))
				stats.Record(ctx, metrics.TipSetHeight.M(int64(ts.Height())))
				stats.Record(ctx, metrics.ExtractDuration.M(metrics.SinceInMilliseconds(estart)))
			}
			log.Infow("done tipset extracting", "tsk", tsk, "height", ts.Height(), "elapsed", time.Now().Sub(estart).String())
		}
	}
}

func (r *RaCailum) AlertOutdatedFinalHeight(ctx context.Context, outdatedGap int64) {
	tick := time.NewTicker(30 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			finalHeight, err := r.activeSeg.GetFinalHeight(ctx)
			if err != nil {
				log.Errorf("get final height failed: %v", err)
				stats.Record(ctx, metrics.OutdatedFinalHeight.M(1))
				continue
			}

			curEpoch := common.GetCurEpoch()
			if curEpoch-finalHeight >= abi.ChainEpoch(outdatedGap) {
				log.Warnf("finalHeight %v lag behind curEpoch %v more than outdatedGap %v", finalHeight, curEpoch, outdatedGap)
				stats.Record(ctx, metrics.OutdatedFinalHeight.M(1))
				continue
			}

			log.Infof("finalHeight %v not lag behind curEpoch %v more than outdatedGap %v", finalHeight, curEpoch, outdatedGap)
			stats.Record(ctx, metrics.OutdatedFinalHeight.M(0))
		case <-ctx.Done():
			return
		}
	}
}

// Grafana returns the instance of *Grafana
// func (r *RaCailum) Grafana() *grafana.Grafana {
//     return r.gr
// }
