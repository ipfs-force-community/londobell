package main

import (
	"context"

	"go.uber.org/fx"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"
)

var multiQuerySegmentCmd = &cli.Command{
	Name: "multiquery-segment",
	Subcommands: []*cli.Command{
		segmentUpdateCmd,
		//segmentShowCmd,
	},
}

var segmentUpdateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},

		&cli.StringSliceFlag{
			Name: "dsn-write-slice",
		},
		&cli.BoolFlag{
			Name: "set-active",
		},
	},

	Action: func(cctx *cli.Context) error {
		components := struct {
			fx.In
			Stm *multiquery.StateManager
		}{}

		stopper, err := dix.New(
			cctx.Context,
			multiquery.MultiQuery(context.TODO(), &components),
			multiquery.InjectRepoPath(cctx),
		)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		segname := cctx.String("name")
		slog := log.With("seg", segname)

		setInfo := false
		info, _, err := components.Stm.LoadInfo(segname)
		if err != nil {
			return err
		}

		//多写数据库
		if dsnSlice := cctx.StringSlice("dsn-write-slice"); len(dsnSlice) != 0 {
			setInfo = true
			info.DSN.NewWrite = dsnSlice
		}

		if setInfo {
			if err := components.Stm.SetInfo(segname, info); err != nil {
				return err
			}

			slog.Infof("info set: %#v", info)
		}

		if cctx.Bool("set-active") {
			if err := components.Stm.SetActive(segname); err != nil {
				return err
			}

			slog.Info("activated")
		}

		slog.Info("done")
		return nil
	},
}

//var segmentShowCmd = &cli.Command{
//	Name: "show",
//	Flags: []cli.Flag{
//		&cli.StringFlag{
//			Name:     "name",
//			Required: true,
//		},
//
//		&cli.BoolFlag{
//			Name:  "info",
//			Value: true,
//		},
//
//		&cli.BoolFlag{
//			Name:  "boundary",
//			Value: true,
//		},
//
//		&cli.BoolFlag{
//			Name:  "active",
//			Value: true,
//		},
//	},
//	Action: func(cctx *cli.Context) error {
//		api, _, err := GetAPIV0(cctx)
//		if err != nil {
//			return err
//		}
//		segname := cctx.String("name")
//		slog := log.With("seg", segname)
//		detail, err := api.SegmentDetail(cctx.Context, segname)
//		if err != nil {
//			return err
//		}
//		if cctx.Bool("info") {
//			if detail.Info != nil {
//				slog.Infow("info", "dns-write-slice", detail.Info.DSN.NewWrite, "dns-read", detail.Info.DSN.Read)
//			} else {
//				slog.Info("segment info not found")
//			}
//		}
//		if cctx.Bool("boundary") {
//			if detail.Boundary != nil {
//				bound := detail.Boundary
//				slog.Infow("boundary", "hi-epoch", bound.Hi.Epoch, "hi-tsk", bound.Hi.TSK.String(), "lo-epoch", bound.Lo.Epoch, "lo-tsk", bound.Lo.TSK)
//			} else {
//				slog.Info("segment boundary not found")
//			}
//		}
//		if cctx.Bool("active") {
//			if detail.Active == "" {
//				slog.Infof("active segment not found")
//			} else {
//				slog.Infof("current active: %s", detail.Active)
//			}
//		}
//		return nil
//	},
//}
