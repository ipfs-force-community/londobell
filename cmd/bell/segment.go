package main

import (
	"fmt"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/racailum/segment"
)

var segmentCmd = &cli.Command{
	Name: "segment",
	Subcommands: []*cli.Command{
		segmentUpdateCmd,
		segmentShowCmd,
	},
}

var segmentUpdateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},

		&cli.StringFlag{
			Name:   "dsn-write",
			Hidden: true,
		},

		&cli.StringSliceFlag{
			Name: "dsn-write-slice",
		},

		&cli.StringFlag{
			Name: "dsn-read",
		},

		&cli.StringFlag{
			Name: "child-hi",
		},

		&cli.StringFlag{
			Name: "child-lo",
		},

		&cli.BoolFlag{
			Name: "set-active",
		},
		&cli.BoolFlag{
			Name:  "local",
			Value: true,
			Usage: "local or remote",
		},
	},

	Action: func(cctx *cli.Context) error {
		di := struct {
			fx.In
			SegMgr *segment.Manager
			CStore common.ChainStore
		}{}

		stopper, err := dix.New(
			cctx.Context,
			dep.WalkRaCalium(cctx, fxlog, &di),
			dep.InjectRepoPath(cctx),
		)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		segname := cctx.String("name")
		slog := log.With("seg", segname)

		setInfo := false
		info, _, err := di.SegMgr.LoadInfo(segname)
		if err != nil {
			return err
		}

		if dsn := cctx.String("dsn-write"); dsn != "" {
			setInfo = true
			info.DSN.Write = dsn
		}

		//多写数据库
		if dsnSlice := cctx.StringSlice("dsn-write-slice"); len(dsnSlice) != 0 {
			setInfo = true
			info.DSN.NewWrite = dsnSlice
		}

		if cctx.IsSet("dsn-read") {
			setInfo = true
			info.DSN.Read = cctx.String("dsn-read")
		}

		if setInfo {
			if err := di.SegMgr.SetInfo(segname, info); err != nil {
				return err
			}

			slog.Infof("info set: %#v", info)
		}

		if cctx.IsSet("child-hi") || cctx.IsSet("child-lo") {
			if err := setSegmentBoundary(cctx, slog, di.CStore, segname, di.SegMgr); err != nil {
				return err
			}
		}

		if cctx.Bool("set-active") {
			if err := di.SegMgr.SetActive(segname); err != nil {
				return err
			}

			slog.Info("activated")
		}

		slog.Info("done")
		return nil
	},
}

func setSegmentBoundary(cctx *cli.Context, slog *zap.SugaredLogger, cstore common.ChainStore, segname string, segmgr *segment.Manager) error {
	bound, _, err := segmgr.LoadBoundary(segname)
	if err != nil {
		return fmt.Errorf("load boundary for %s: %w", segname, err)
	}

	set := false
	if s := cctx.String("child-hi"); s != "" {
		tsk, err := parsetTipSetKey(s)
		if err != nil {
			return fmt.Errorf("hi-child key: %w", err)
		}

		hi, err := common.LoadLinkedTipSet(cstore, tsk)
		if err != nil {
			return fmt.Errorf("load hi tipset: %w", err)
		}

		set = true
		bound.SetHi(hi)
	}

	if s := cctx.String("child-lo"); s != "" {
		tsk, err := parsetTipSetKey(s)
		if err != nil {
			return fmt.Errorf("lo-child key: %w", err)
		}

		lo, err := common.LoadLinkedTipSet(cstore, tsk)
		if err != nil {
			return fmt.Errorf("load lo tipset: %w", err)
		}

		set = true
		bound.SetLo(lo)
	}

	if set {
		if bound.Hi.Epoch < bound.Lo.Epoch {
			return fmt.Errorf("hi-epoch(%d) lower than lo-epoch(%d)", bound.Hi.Epoch, bound.Lo.Epoch)
		}

		if err := segmgr.SetBoundary(segname, bound); err != nil {
			return err
		}

		slog.Infow("boundary set", "hi-epoch", bound.Hi.Epoch, "hi-tsk", bound.Hi.TSK.String(), "lo-epoch", bound.Lo.Epoch, "lo-tsk", bound.Lo.TSK)
	}

	return nil
}

var segmentShowCmd = &cli.Command{
	Name: "show",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},

		&cli.BoolFlag{
			Name:  "info",
			Value: true,
		},

		&cli.BoolFlag{
			Name:  "boundary",
			Value: true,
		},

		&cli.BoolFlag{
			Name:  "active",
			Value: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		api, _, err := GetAPIV0(cctx)
		if err != nil {
			return err
		}
		segname := cctx.String("name")
		slog := log.With("seg", segname)
		detail, err := api.SegmentDetail(cctx.Context, segname)
		if err != nil {
			return err
		}
		if cctx.Bool("info") {
			if detail.Info != nil {
				slog.Infow("info", "dns-write-slice", detail.Info.DSN.NewWrite, "dns-read", detail.Info.DSN.Read)
			} else {
				slog.Info("segment info not found")
			}
		}
		if cctx.Bool("boundary") {
			if detail.Boundary != nil {
				bound := detail.Boundary
				slog.Infow("boundary", "hi-epoch", bound.Hi.Epoch, "hi-tsk", bound.Hi.TSK.String(), "lo-epoch", bound.Lo.Epoch, "lo-tsk", bound.Lo.TSK)
			} else {
				slog.Info("segment boundary not found")
			}
		}
		if cctx.Bool("active") {
			if detail.Active == "" {
				slog.Infof("active segment not found")
			} else {
				slog.Infof("current active: %s", detail.Active)
			}
		}
		return nil
	},
}
