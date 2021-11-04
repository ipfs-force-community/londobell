package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/prometheus"
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

func buildRaApp(cctx *cli.Context, full v0api.FullNode, target interface{}) (dix.StopFunc, error) {
	return dix.New(cctx.Context,
		dep.Bell(cctx.Context, fxlog, target),
		dix.Override(new(v0api.FullNode), full),
	)

}

var raRunCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "prometheus-port",
			Usage: "specify the port of prometheus",
			Value: "2112",
		},
		&cli.BoolFlag{
			Name:  "prometheus",
			Usage: "choose whether use prometheus",
			Value: true,
		},
	},
	Action: func(cctx *cli.Context) error {
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

		defer stopper(cctx.Context)

		ctx := cctx.Context

		ch, err := components.Notifier.Sub(ctx)
		if err != nil {
			return fmt.Errorf("sub head change: %w", err)
		}

		if cctx.Bool("prometheus") {
			go prometheus.Run(cctx.String("prometheus-port"))
		}

		go func() {
			log.Info("serving http pprof")
			if err := http.ListenAndServe("0.0.0.0:56060", nil); err != nil {
				log.Errorf("serving http pprof: %s", err)
			}
		}()

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

				ts, err := components.CS.LoadTipSet(tsk)
				if err != nil {
					log.Errorf("failed to load tipset %s: %s", tsk, err)
					continue HEAD_LOOP
				}

				log.Infow("incoming tipset", "tsk", tsk, "height", ts.Height())
				estart := time.Now()
				if err := components.Ra.Extract(ctx, ts); err != nil {
					log.Errorf("failed to persist tipset: %s", err)
				}
				log.Infow("done tipset extracting", "tsk", tsk, "height", ts.Height(), "elapsed", time.Now().Sub(estart).String())
			}
		}
	},
}
