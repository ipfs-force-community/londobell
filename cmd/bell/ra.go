package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/dep"
	"github.com/dtynn/londobell/lib/fxex"
	"github.com/dtynn/londobell/racailum"
	"github.com/dtynn/londobell/racailum/grafana"
	"github.com/dtynn/londobell/racailum/segment/aggregate"
)

var raCmd = &cli.Command{
	Name: "ra",
	Subcommands: []*cli.Command{
		raRunCmd,
		raExtractCmd,
		raAggregateCmd,
		raDryCmd,
		raGrafanaCmd,
	},
}

var (
	raFlagSegName = &cli.StringFlag{
		Name:     "seg-name",
		Required: true,
	}

	raFlagSegDSN = &cli.StringFlag{
		Name:     "seg-dsn",
		Required: true,
	}

	raFlagSegDSNRead = &cli.StringFlag{
		Name:     "seg-dsn-read",
		Required: true,
	}

	raFlagAggDir = &cli.StringFlag{
		Name: "agg-dir",
	}

	raFlagGasTracing = &cli.BoolFlag{
		Name: "gas-tracing",
	}

	raFlagGrafanaDir = &cli.StringFlag{
		Name: "grafana-dir",
	}

	raFlagGrafanaListen = &cli.BoolFlag{
		Name: "grafana-listen",
	}

	raFlagEnableGrafana = &cli.BoolFlag{
		Name: "enable-grafana",
	}

	raFlags = []cli.Flag{
		dep.FlagMgoMetaMgrDSN,

		raFlagSegName,

		raFlagSegDSN,
		raFlagSegDSNRead,

		raFlagAggDir,

		raFlagGasTracing,

		raFlagEnableGrafana,
		raFlagGrafanaDir,
		raFlagGrafanaListen,
	}
)

func copyFlags(src []cli.Flag) []cli.Flag {
	dst := make([]cli.Flag, len(src))
	copy(dst, src)
	return dst
}

func buildRaApp(cctx *cli.Context, target interface{}) (*fx.App, error) {
	aggOpt, err := aggregate.DefaultOptions()
	if err != nil {
		return nil, fmt.Errorf("default aggregate options: %w", err)
	}

	if cctx.IsSet(raFlagAggDir.Name) {
		aggOpt.Dir = cctx.String(raFlagAggDir.Name)
	}

	grOpt, err := grafana.DefaultOptions()
	if err != nil {
		return nil, fmt.Errorf("default grafana options: %w", err)
	}

	if cctx.IsSet(raFlagGrafanaDir.Name) {
		grOpt.ScriptDir = cctx.String(raFlagAggDir.Name)
	}

	if cctx.IsSet(raFlagGrafanaDir.Name) {
		grOpt.ScriptDir = cctx.String(raFlagAggDir.Name)
	}

	return dep.BellApp(
		cctx.Context,
		fxlog,
		target,
		fxex.ProvideEx(
			racailum.Config{
				Aggregate:        aggOpt,
				Grafana:          grOpt,
				EnableGasTracing: cctx.IsSet(raFlagGasTracing.Name),
				EnableGrafana:    cctx.IsSet(raFlagEnableGrafana.Name),
				Segments: []racailum.SegmentConfig{
					{
						DSN:     cctx.String(raFlagSegDSN.Name),
						Name:    cctx.String(raFlagSegName.Name),
						ReadDSN: cctx.String(raFlagSegDSNRead.Name),
					},
				},
			},
			dep.MgoMetaMgrDSN(cctx.String(dep.FlagMgoMetaMgrDSN.Name)),
		),
	), nil
}

var raRunCmd = &cli.Command{
	Name:  "run",
	Flags: raFlags,
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS       common.ChainStore
			Notifier common.HeadNotifier
			Ra       *racailum.RaCailum
		}

		app, err := buildRaApp(cctx, &components)
		if err != nil {
			return err
		}

		err = app.Start(cctx.Context)
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
	Flags: append(copyFlags(raFlags),
		&cli.StringFlag{
			Name:     "dest-child",
			Required: true,
		},
	),
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS common.ChainStore
			Ra *racailum.RaCailum
		}

		app, err := buildRaApp(cctx, &components)
		if err != nil {
			return err
		}

		err = app.Start(cctx.Context)
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

var raAggregateCmd = &cli.Command{
	Name:  "agg",
	Usage: "start aggregations within given epoch range manually",
	Flags: append(copyFlags(raFlags),
		&cli.StringFlag{
			Name:     "hi-child",
			Required: true,
		},

		&cli.StringFlag{
			Name:     "lo-child",
			Required: true,
		},
	),
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS common.ChainStore
			Ra *racailum.RaCailum
		}

		app, err := buildRaApp(cctx, &components)
		if err != nil {
			return err
		}

		err = app.Start(cctx.Context)
		if err != nil {
			return err
		}

		defer app.Stop(cctx.Context)
		log.Info("ra app constructed")

		var hi, lo *common.LinkedTipSet
		{
			tsk, err := parsetTipSetKey(cctx.String("hi-child"))
			if err != nil {
				return fmt.Errorf("hi-child key: %w", err)
			}

			hi, err = common.LoadLinkedTipSet(components.CS, tsk)
			if err != nil {
				return fmt.Errorf("load hi tipset: %w", err)
			}
		}

		{
			tsk, err := parsetTipSetKey(cctx.String("lo-child"))
			if err != nil {
				return fmt.Errorf("lo-child key: %w", err)
			}

			lo, err = common.LoadLinkedTipSet(components.CS, tsk)
			if err != nil {
				return fmt.Errorf("load lo tipset: %w", err)
			}
		}

		if hi == nil || lo == nil {
			log.Warn("both boundaries are required")
			return nil
		}

		log.Infow("boundry loaded", "lo", lo.Height(), "hi", hi.Height())

		err = components.Ra.Aggregate(cctx.Context, lo.TipSet, hi.TipSet)
		if err != nil {
			return err
		}

		return nil
	},
}

var raDryCmd = &cli.Command{
	Name:  "dry",
	Usage: "run a dry extracting",
	Flags: append(copyFlags(raFlags),
		&cli.StringFlag{
			Name:     "child",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "col",
			Required: true,
		},
		&cli.IntFlag{
			Name:  "count",
			Value: 10,
		},
	),
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS common.ChainStore
			Ra *racailum.RaCailum
		}

		app, err := buildRaApp(cctx, &components)
		if err != nil {
			return err
		}

		err = app.Start(cctx.Context)
		if err != nil {
			return err
		}

		defer app.Stop(cctx.Context)
		log.Info("ra app constructed")

		tsk, err := parsetTipSetKey(cctx.String("child"))
		if err != nil {
			return fmt.Errorf("child key: %w", err)
		}

		ts, err := common.LoadLinkedTipSet(components.CS, tsk)
		if err != nil {
			return fmt.Errorf("load tipset: %w", err)
		}

		log.Infow("ts loaded", "height", ts.Height())

		res, err := components.Ra.DryState(cctx.Context, ts)
		if err != nil {
			return err
		}

		log.Infow("results loaded", "count", len(res))

		wanted := cctx.String("col")
		wantedCount := cctx.Int("count")
		got := 0

		dlog := log.With("col", wanted)

	WANT_LOOP:
		for ri := range res {
			for di := range res[ri].Docs {
				if res[ri].Docs[di].CollectionName() == wanted {
					got++
					doc := res[ri].Docs[di]

					if p, ok := doc.(common.DetailPrinter); ok {
						p.PrintDetail(dlog)
					} else {
						dlog.Infof("%#v", doc)
					}

					if got >= wantedCount {
						break WANT_LOOP
					}
				}
			}
		}

		log.Infow("done", "got", got)

		return nil
	},
}

var raGrafanaCmd = &cli.Command{
	Name:  "grafana",
	Flags: append(copyFlags(raFlags)),
	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			Ra *racailum.RaCailum
		}

		app, err := buildRaApp(cctx, &components)
		if err != nil {
			return err
		}

		err = app.Start(cctx.Context)
		if err != nil {
			return err
		}

		defer app.Stop(cctx.Context)

		srv := components.Ra.Grafana().HTTPServer(cctx.Context)

		log.Infof("grafana listen on %s", srv.Addr)
		return srv.ListenAndServe()
	},
}
