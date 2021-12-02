package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/node"
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
		ctx, cancel := context.WithCancel(cctx.Context)
		defer cancel()

		var components struct {
			fx.In
			CS       common.ChainStore
			Notifier common.HeadNotifier
			Ra       *racailum.RaCailum
			Cfg      racailum.Config
			Mux      *http.ServeMux
		}

		stopper, err := dix.New(
			ctx,
			dep.Bell(ctx, fxlog, &components),
			dep.InjectRepoPath(cctx),
			dep.InjectFullNode(cctx),
		)
		if err != nil {
			return err
		}

		httpStoper, errCh := serveHTTP(components.Cfg.HTTP.Listen, components.Mux)
		select {
		case err = <-errCh:

		case <-time.After(time.Duration(components.Cfg.HTTP.StableWait)):

		}

		if err != nil {
			return fmt.Errorf("start http server: %w", err)
		}

		doneCh := node.MonitorShutdown(
			ctx.Done(),
			node.ShutdownHandler{Component: "http server", StopFunc: httpStoper},
			node.ShutdownHandler{Component: "application", StopFunc: node.StopFunc(stopper)},
		)

		ch, err := components.Notifier.Sub(ctx)
		if err != nil {
			return fmt.Errorf("sub head change: %w", err)
		}

		// TODO: turn this code block into some Run func
	HEAD_LOOP:
		for {
			select {
			case <-doneCh:
				log.Info("quit head-change loop")
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
					stats.Record(ctx, metrics.ExtractError.M(1))
				} else {
					stats.Record(ctx, metrics.ExtractError.M(0))
					stats.Record(ctx, metrics.TipSetHeight.M(int64(ts.Height())))
					stats.Record(ctx, metrics.ExtractDuration.M(metrics.SinceInMilliseconds(estart)))
				}
				log.Infow("done tipset extracting", "tsk", tsk, "height", ts.Height(), "elapsed", time.Now().Sub(estart).String())
			}
		}
	},
}

func serveHTTP(addr string, mux *http.ServeMux) (func(context.Context) error, <-chan error) {
	errCh := make(chan error, 1)
	if addr == "" {
		close(errCh)
		log.Warn("no listen address provided")
		return func(context.Context) error { return nil }, errCh
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		defer close(errCh)

		log.Infof("http server will start on %s", addr)
		err := srv.ListenAndServe()
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}

		return
	}()

	return srv.Shutdown, errCh
}
