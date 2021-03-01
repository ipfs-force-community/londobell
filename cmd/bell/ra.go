package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/dep"
	"github.com/dtynn/londobell/lib/fxex"
	"github.com/dtynn/londobell/racailum"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

var raCmd = &cli.Command{
	Name: "ra",
	Subcommands: []*cli.Command{
		raRunCmd,
		raExtractCmd,
	},
}

var raRunCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		dep.FlagMgoBstoreDSN,
		dep.FlagMgoMetaDSDSN,
		dep.FlagMgoMetaMgrDSN,

		&cli.StringFlag{
			Name:     "seg-name",
			Required: true,
		},

		&cli.StringFlag{
			Name:     "seg-dsn",
			Required: true,
		},

		&cli.BoolFlag{
			Name: "gas-tracing",
		},
	},
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS       common.ChainStore
			Notifier common.HeadNotifier
			Ra       *racailum.RaCailum
		}

		app := dep.BellApp(
			cctx.Context,
			fxlog,
			&components,
			fxex.ProvideEx(
				racailum.Config{
					EnableGasTracing: cctx.IsSet("gas-tracing"),
					Segments: []racailum.SegmentConfig{
						{
							DSN:  cctx.String("seg-dsn"),
							Name: cctx.String("seg-name"),
						},
					},
				},
				dep.MgoBstoreDSN(cctx.String(dep.FlagMgoBstoreDSN.Name)),
				dep.MgoMetaDSDSN(cctx.String(dep.FlagMgoMetaDSDSN.Name)),
				dep.MgoMetaMgrDSN(cctx.String(dep.FlagMgoMetaMgrDSN.Name)),
			),
		)

		err := app.Start(cctx.Context)
		if err != nil {
			return err
		}

		defer app.Stop(cctx.Context)

		ctx := cctx.Context

		ch, err := components.Notifier.Sub(ctx)
		if err != nil {
			return fmt.Errorf("sub head change: %w", err)
		}

		go func() {
			log.Info("serving http pprof")
			if err := http.ListenAndServe("127.0.0.1:56060", nil); err != nil {
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

var raExtractCmd = &cli.Command{
	Name: "extract",
	Flags: []cli.Flag{
		dep.FlagMgoBstoreDSN,
		dep.FlagMgoMetaDSDSN,
		dep.FlagMgoMetaMgrDSN,

		&cli.StringFlag{
			Name:     "seg-name",
			Required: true,
		},

		&cli.StringFlag{
			Name:     "seg-dsn",
			Required: true,
		},

		&cli.StringFlag{
			Name:     "dest-child",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS common.ChainStore
			Ra *racailum.RaCailum
		}

		app := dep.BellApp(
			cctx.Context,
			fxlog,
			&components,
			fxex.ProvideEx(
				racailum.Config{
					Segments: []racailum.SegmentConfig{
						{
							DSN:  cctx.String("seg-dsn"),
							Name: cctx.String("seg-name"),
						},
					},
				},
				dep.MgoBstoreDSN(cctx.String(dep.FlagMgoBstoreDSN.Name)),
				dep.MgoMetaDSDSN(cctx.String(dep.FlagMgoMetaDSDSN.Name)),
				dep.MgoMetaMgrDSN(cctx.String(dep.FlagMgoMetaMgrDSN.Name)),
			),
		)

		err := app.Start(cctx.Context)
		if err != nil {
			return err
		}

		defer app.Stop(cctx.Context)

		tsk, err := parsetTipSetKey(cctx.String("dest-child"))
		if err != nil {
			return fmt.Errorf("dest-child key: %w", err)
		}

		ts, err := common.LoadLinkedTipSet(components.CS, tsk)
		if err != nil {
			return fmt.Errorf("load tipset: %w", err)
		}

		start := time.Now()
		log.Infow("attempt to extract form given tipset", "tsk", ts.Key(), "epoch", ts.Height())
		err = components.Ra.Extract(cctx.Context, ts.TipSet)
		if err != nil {
			return err
		}

		log.Infow("done extracting", "elapsed", time.Now().Sub(start).String())
		return nil
	},
}
