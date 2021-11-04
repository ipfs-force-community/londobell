package main

import (
	"fmt"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/api/v0api"

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
			Name: "dsn-write",
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
	},

	Action: func(cctx *cli.Context) error {
		di := struct {
			fx.In
			SegMgr *segment.Manager
			Full   v0api.FullNode
			CStore common.ChainStore
		}{}

		stopper, err := dix.New(
			cctx.Context,
			dep.Bell(cctx.Context, fxlog, &di),
			dep.InjectRepoPath(cctx),
			dep.InjectFullNode(cctx),
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
			if err := setSegmentBoundary(cctx, slog, di.Full, di.CStore, segname, di.SegMgr); err != nil {
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

func setSegmentBoundary(cctx *cli.Context, slog *zap.SugaredLogger, full v0api.FullNode, cstore common.ChainStore, segname string, segmgr *segment.Manager) error {
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
		di := struct {
			fx.In
			SegMgr *segment.Manager
		}{}

		stopper, err := dix.New(
			cctx.Context,
			dep.Bell(cctx.Context, fxlog, &di),
			dep.InjectRepoPath(cctx),
		)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		segname := cctx.String("name")
		slog := log.With("seg", segname)

		if cctx.Bool("info") {
			info, has, err := di.SegMgr.LoadInfo(segname)
			if err != nil {
				return err
			}

			if has {
				slog.Infof("info: %#v", info)
			} else {
				slog.Info("info not found")
			}
		}

		if cctx.Bool("boundary") {
			bound, has, err := di.SegMgr.LoadBoundary(segname)
			if err != nil {
				return err
			}

			if has {
				slog.Infow("boundary", "hi-epoch", bound.Hi.Epoch, "hi-tsk", bound.Hi.TSK.String(), "lo-epoch", bound.Lo.Epoch, "lo-tsk", bound.Lo.TSK)
			} else {
				slog.Info("boundary not found")
			}
		}

		if cctx.Bool("active") {
			activated, has, err := di.SegMgr.LoadActive()
			if err != nil {
				return err
			}

			if !has {
				slog.Info("active segment not set")
			} else {
				extra := ""
				if activated != segname {
					extra = fmt.Sprintf("(%s)", activated)
				}

				slog.Infof("current segment activated: %v%s", activated == segname, extra)
			}
		}
		return nil
	},
}
