package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dtynn/dix"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/ipfs-force-community/londobell/api"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/racailum"
	"github.com/ipfs-force-community/londobell/tmpbell"
)

var daemonCmd = &cli.Command{
	Name: "daemon",
	Subcommands: []*cli.Command{
		daemonStartCmd,
		daemonStopCmd,
	},
	Usage: "start/stop bell sync chain data",
}

var daemonStartCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "tmp",
			Usage: "enable temporary db to store close data",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		shutdownCh := make(chan struct{})

		var components struct {
			fx.In
			NodeAPI  api.BellNodeAPI
			Cfg      racailum.Config
			Mux      *http.ServeMux
			Notifier common.HeadNotifier
			Ra       *racailum.RaCailum
			Tmp      *tmpbell.TmpBell
		}

		stopper, err := dix.New(ctx,
			dep.Bell(ctx, fxlog, &components),
			dep.InjectFullNode(cctx),
			dep.InjectRepoPath(cctx),
			dix.Override(new(dtypes.ShutdownChan), shutdownCh),
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
			shutdownCh,
			node.ShutdownHandler{Component: "http server", StopFunc: httpStoper},
			node.ShutdownHandler{Component: "application", StopFunc: node.StopFunc(stopper)},
		)

		if cctx.Bool("tmp") {
			// read config for tmp db
			tempDBCapacity, err := GetTempDBCapacity(cctx)
			if err != nil {
				return fmt.Errorf("get temp db capacity failed: %v", err)
			}

			triggerSpan, err := GetTriggerSpan(cctx)
			if err != nil {
				return fmt.Errorf("get triggerSpan from config failed: %v", err)
			}

			// temporary db
			err = components.Tmp.MonitorForTmpDB(ctx, tempDBCapacity, triggerSpan)
			if err != nil {
				return fmt.Errorf("failed to activate temporary db: %v", err)
			}
			go components.Tmp.AlertOutdatedFinalHeight(ctx, components.Cfg.OutdatedGap)
		} else {
			ch, err := components.Notifier.Sub(ctx)
			if err != nil {
				return fmt.Errorf("sub head change: %w", err)
			}
			go components.Ra.Run(ctx, doneCh, ch)
			go components.Ra.AlertOutdatedFinalHeight(ctx, components.Cfg.OutdatedGap)
		}

		addr := components.Cfg.HTTP.RPCListen
		if addr == "" {
			addr = racailum.DefaultRPCListenAddr
		}
		endpoint, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return fmt.Errorf("parse addr: %s, err: %v", addr, err)
		}

		return ServeRPC(&components.NodeAPI, stopper, endpoint, doneCh, 0)
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

var daemonStopCmd = &cli.Command{
	Name: "stop",
	Action: func(c *cli.Context) error {
		api, _, err := GetAPIV0(c)
		if err != nil {
			return err
		}
		return api.ShutDown(c.Context)
	},
}
