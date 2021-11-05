package main

import (
	"fmt"
	"time"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum"
)

var raCmd = &cli.Command{
	Name: "ra",
	Subcommands: []*cli.Command{
		raRunCmd,
		// raExtractCmd,
		// raAggregateCmd,
		// raDryCmd,
		// raGrafanaCmd,
	},
}

var raRunCmd = &cli.Command{
	Name: "run",
	Action: func(cctx *cli.Context) error {
		rpath, err := dep.GetRepoPath(cctx)
		if err != nil {
			return err
		}
		cfg, err := dep.LoadRaConfig(rpath)
		if err != nil {
			return err
		}
		var components struct {
			fx.In
			CS       common.ChainStore
			Notifier common.HeadNotifier
			Ra       *racailum.RaCailum
		}

		stopper, err := dix.New(
			cctx.Context,
			dep.Bell(cctx.Context, fxlog, &components),
			dep.InjectRepoPath(cctx),
			dep.InjectFullNode(cctx),
		)
		if err != nil {
			return err
		}
		defer stopper(cctx.Context) // nolint: errcheck

		ctx := cctx.Context

		ch, err := components.Notifier.Sub(ctx)
		if err != nil {
			return fmt.Errorf("sub head change: %w", err)
		}

		if err := setupMetrics(cfg.Metrics); err != nil {
			return fmt.Errorf("setup metrics err: %v", err)
		}

	HEAD_LOOP:
		for {
			select {
			case <-ctx.Done():
				log.Info("context done")
				return nil

			case tsk, ok := <-ch:
				if !ok {
					log.Warn("tsk chan closed")
					return nil
				}
				lstart := time.Now()
				ts, err := components.CS.LoadTipSet(tsk)
				stats.Record(ctx, metrics.LoadTipSetDuration.M(metrics.SinceInMilliseconds(lstart)))
				if err != nil {
					log.Errorf("failed to load tipset %s: %s", tsk, err)
					continue HEAD_LOOP
				}

				log.Infow("incoming tipset", "tsk", tsk, "height", ts.Height())
				estart := time.Now()
				if err := components.Ra.Extract(ctx, ts); err != nil {
					log.Errorf("failed to persist tipset: %s", err)
					metrics.RecordInc(ctx, metrics.ExtractError)
				} else {
					metrics.RecordInc(ctx, metrics.ExtractComplete)
					stats.Record(ctx, metrics.TipSetHeight.M(int64(ts.Height())))
					stats.Record(ctx, metrics.ExtractDuration.M(metrics.SinceInMilliseconds(estart)))
				}
				log.Infow("done tipset extracting", "tsk", tsk, "height", ts.Height(), "elapsed", time.Now().Sub(estart).String())
			}
		}
	},
}
