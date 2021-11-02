package main

import (
	"fmt"
	"path/filepath"

	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/dep"
	"github.com/dtynn/londobell/lib/fxex"
	"github.com/dtynn/londobell/racailum/segment"
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

		ds, err := openSegmentDS(cctx)
		if err != nil {
			return err
		}

		segmgr, err := segment.NewManager(ds)
		if err != nil {
			return err
		}

		segname := cctx.String("name")
		slog := log.With("seg", segname)

		setInfo := false
		info, _, err := segmgr.LoadInfo(segname)
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
			if err := segmgr.SetInfo(segname, info); err != nil {
				return err
			}

			slog.Infof("info set: %#v", info)
		}

		if cctx.IsSet("child-hi") || cctx.IsSet("child-lo") {
			if err := setSegmentBoundary(cctx, slog, segname, segmgr); err != nil {
				return err
			}
		}

		if cctx.Bool("set-active") {
			if err := segmgr.SetActive(segname); err != nil {
				return err
			}

			slog.Info("activated")
		}

		slog.Info("done")
		return nil
	},
}

func setSegmentBoundary(cctx *cli.Context, slog *zap.SugaredLogger, segname string, segmgr *segment.Manager) error {
	full, closer, err := getFullNode(cctx)
	if err != nil {
		return err
	}

	defer closer()

	var components struct {
		fx.In
		CS common.ChainStore
	}

	app := dep.BellApp(
		cctx.Context,
		fxlog,
		&components,
		fxex.ProvideEx(
			fxex.As(full, new(v0api.FullNode)),
		),
	)

	err = app.Start(cctx.Context)
	if err != nil {
		return err
	}

	defer app.Stop(cctx.Context) // nolint: errcheck

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

		hi, err := common.LoadLinkedTipSet(components.CS, tsk)
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

		lo, err := common.LoadLinkedTipSet(components.CS, tsk)
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
		ds, err := openSegmentDS(cctx)
		if err != nil {
			return err
		}

		segmgr, err := segment.NewManager(ds)
		if err != nil {
			return err
		}

		segname := cctx.String("name")
		slog := log.With("seg", segname)

		if cctx.Bool("info") {
			info, has, err := segmgr.LoadInfo(segname)
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
			bound, has, err := segmgr.LoadBoundary(segname)
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
			activated, has, err := segmgr.LoadActive()
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

func segmentDSPath(cctx *cli.Context) (string, error) {
	dir, err := getRepoHomeDir(cctx)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "segment"), nil
}

func openSegmentDS(cctx *cli.Context) (dtypes.MetadataDS, error) {
	dsPath, err := segmentDSPath(cctx)
	if err != nil {
		return nil, err
	}

	return levelDs(dsPath, false)
}

func levelDs(path string, readonly bool) (dtypes.MetadataDS, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}
