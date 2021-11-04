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

	// "github.com/ipfs-force-community/londobell/racailum/grafana"
	"github.com/ipfs-force-community/londobell/racailum/segment"
	// "github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
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
	racfg, err := loadConfig(cctx)
	if err != nil {
		return nil, err
	}

	segds, err := openSegmentDS(cctx)
	if err != nil {
		return nil, err
	}

	segmgr, err := segment.NewManager(segds)
	if err != nil {
		return nil, err
	}

	return dix.New(cctx.Context,
		dep.Bell(cctx.Context, fxlog, target),
		dix.Override(new(v0api.FullNode), full),
		dix.Override(new(racailum.Config), racfg),
		dix.Override(new(*segment.Manager), segmgr),
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
		full, closer, err := getFullNode(cctx)
		if err != nil {
			return err
		}

		defer closer()

		var components struct {
			fx.In
			CS       common.ChainStore
			Notifier common.HeadNotifier
			Ra       *racailum.RaCailum
		}

		stopper, err := buildRaApp(cctx, full, &components)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

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

// var raExtractCmd = &cli.Command{
//     Name: "extract",
//     Flags: append(copyFlags(raFlags),
//         &cli.StringFlag{
//             Name:     "dest-child",
//             Required: true,
//         },
//     ),
//     Action: func(cctx *cli.Context) error {
//         var components struct {
//             fx.In
//             CS common.ChainStore
//             Ra *racailum.RaCailum
//         }

//         app, err := buildRaApp(cctx, &components)
//         if err != nil {
//             return err
//         }

//         err = app.Start(cctx.Context)
//         if err != nil {
//             return err
//         }

//         defer app.Stop(cctx.Context)

//         tsk, err := parsetTipSetKey(cctx.String("dest-child"))
//         if err != nil {
//             return fmt.Errorf("dest-child key: %w", err)
//         }

//         ts, err := common.LoadLinkedTipSet(components.CS, tsk)
//         if err != nil {
//             return fmt.Errorf("load tipset: %w", err)
//         }

//         start := time.Now()
//         log.Infow("attempt to extract form given tipset", "tsk", ts.Key(), "epoch", ts.Height())
//         err = components.Ra.Extract(cctx.Context, ts.TipSet)
//         if err != nil {
//             return err
//         }

//         log.Infow("done extracting", "elapsed", time.Now().Sub(start).String())
//         return nil
//     },
// }

// var raAggregateCmd = &cli.Command{
//     Name:  "agg",
//     Usage: "start aggregations within given epoch range manually",
//     Flags: append(copyFlags(raFlags),
//         &cli.StringFlag{
//             Name:     "hi-child",
//             Required: true,
//         },

//         &cli.StringFlag{
//             Name:     "lo-child",
//             Required: true,
//         },
//     ),
//     Action: func(cctx *cli.Context) error {
//         var components struct {
//             fx.In
//             CS common.ChainStore
//             Ra *racailum.RaCailum
//         }

//         app, err := buildRaApp(cctx, &components)
//         if err != nil {
//             return err
//         }

//         err = app.Start(cctx.Context)
//         if err != nil {
//             return err
//         }

//         defer app.Stop(cctx.Context)
//         log.Info("ra app constructed")

//         var hi, lo *common.LinkedTipSet
//         {
//             tsk, err := parsetTipSetKey(cctx.String("hi-child"))
//             if err != nil {
//                 return fmt.Errorf("hi-child key: %w", err)
//             }

//             hi, err = common.LoadLinkedTipSet(components.CS, tsk)
//             if err != nil {
//                 return fmt.Errorf("load hi tipset: %w", err)
//             }
//         }

//         {
//             tsk, err := parsetTipSetKey(cctx.String("lo-child"))
//             if err != nil {
//                 return fmt.Errorf("lo-child key: %w", err)
//             }

//             lo, err = common.LoadLinkedTipSet(components.CS, tsk)
//             if err != nil {
//                 return fmt.Errorf("load lo tipset: %w", err)
//             }
//         }

//         if hi == nil || lo == nil {
//             log.Warn("both boundaries are required")
//             return nil
//         }

//         log.Infow("boundray loaded", "lo", lo.Height(), "hi", hi.Height())

//         err = components.Ra.Aggregate(cctx.Context, lo.TipSet, hi.TipSet)
//         if err != nil {
//             return err
//         }

//         return nil
//     },
// }

// var raDryCmd = &cli.Command{
//     Name:  "dry",
//     Usage: "run a dry extracting",
//     Flags: append(copyFlags(raFlags),
//         &cli.StringFlag{
//             Name:     "child",
//             Required: true,
//         },
//         &cli.StringFlag{
//             Name:     "col",
//             Required: true,
//         },
//         &cli.IntFlag{
//             Name:  "count",
//             Value: 10,
//         },
//     ),
//     Action: func(cctx *cli.Context) error {
//         var components struct {
//             fx.In
//             CS common.ChainStore
//             Ra *racailum.RaCailum
//         }

//         app, err := buildRaApp(cctx, &components)
//         if err != nil {
//             return err
//         }

//         err = app.Start(cctx.Context)
//         if err != nil {
//             return err
//         }

//         defer app.Stop(cctx.Context)
//         log.Info("ra app constructed")

//         tsk, err := parsetTipSetKey(cctx.String("child"))
//         if err != nil {
//             return fmt.Errorf("child key: %w", err)
//         }

//         ts, err := common.LoadLinkedTipSet(components.CS, tsk)
//         if err != nil {
//             return fmt.Errorf("load tipset: %w", err)
//         }

//         log.Infow("ts loaded", "height", ts.Height())

//         res, err := components.Ra.DryState(cctx.Context, ts)
//         if err != nil {
//             return err
//         }

//         log.Infow("results loaded", "count", len(res))

//         wanted := cctx.String("col")
//         wantedCount := cctx.Int("count")
//         got := 0

//         dlog := log.With("col", wanted)

//     WANT_LOOP:
//         for ri := range res {
//             for di := range res[ri].Docs {
//                 if res[ri].Docs[di].CollectionName() == wanted {
//                     got++
//                     doc := res[ri].Docs[di]

//                     if p, ok := doc.(common.DetailPrinter); ok {
//                         p.PrintDetail(dlog)
//                     } else {
//                         dlog.Infof("%#v", doc)
//                     }

//                     if got >= wantedCount {
//                         break WANT_LOOP
//                     }
//                 }
//             }
//         }

//         log.Infow("done", "got", got)

//         return nil
//     },
// }

// var raGrafanaCmd = &cli.Command{
//     Name:  "grafana",
//     Flags: append(copyFlags(raFlags)),
//     Action: func(cctx *cli.Context) error {
//         var components struct {
//             fx.In
//             Ra *racailum.RaCailum
//         }

//         app, err := buildRaApp(cctx, &components)
//         if err != nil {
//             return err
//         }

//         err = app.Start(cctx.Context)
//         if err != nil {
//             return err
//         }

//         defer app.Stop(cctx.Context)

//         srv := components.Ra.Grafana().HTTPServer(cctx.Context)

//         log.Infof("grafana listen on %s", srv.Addr)
//         return srv.ListenAndServe()
//     },
// }
