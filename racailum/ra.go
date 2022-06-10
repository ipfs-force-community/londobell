package racailum

import (
	"context"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	lconfig "github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum/grafana"
	"github.com/ipfs-force-community/londobell/racailum/segment"
	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/tracing"
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
		HTTP:             DefaultHTTPOptions(),
		Grafana:          grafana.DefaultOptions(),
		Aggregate:        aggregate.DefaultOptions(),
		Segment:          segment.DefaultOptions(),
		Metrics:          metrics.DefaultOptions(),
		Tracing:          tracing.DefaultOptions(),
		EnableGasTracing: false,
		EnableGrafana:    true,
		EnableDebug:      true,
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
	HTTP             HTTPOptions
	Grafana          grafana.Options
	Aggregate        aggregate.Options
	Segment          segment.Options
	Metrics          metrics.Options
	Tracing          tracing.Options
	EnableGasTracing bool
	EnableGrafana    bool
	EnableDebug      bool
}

// New returns an instance of *RaCailum
func New(ctx context.Context, cfg Config, sub common.HeadNotifier, cs common.ChainStore, stm common.StateManager, segmgr *segment.Manager, shutdownCh dtypes.ShutdownChan) (*RaCailum, error) {
	vm.EnableDetailedTracing = cfg.EnableGasTracing

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
	log.Infow("ra sets sail", "gas-tracing", cfg.EnableGasTracing, "active-seg", activeSegName)
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

// Aggregate is used to trigger aggregations from given tipset boundaries manually
func (r *RaCailum) Aggregate(ctx context.Context, lo, hi *types.TipSet) error {
	loEpoch := lo.Height()
	tss, err := segment.ExtractLinkedTipSets(r.components.cs, hi, &loEpoch)
	if err != nil {
		return fmt.Errorf("load tipsets: %w", err)
	}

	return r.activeSeg.Aggregate(ctx, tss)
}

// DryState runs a dry extraction from given ts
func (r *RaCailum) DryState(ctx context.Context, ts *common.LinkedTipSet) ([]*extract.Res, error) {
	return r.activeSeg.DryExtract(ctx, ts)
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

// Grafana returns the instance of *Grafana
// func (r *RaCailum) Grafana() *grafana.Grafana {
//     return r.gr
// }
