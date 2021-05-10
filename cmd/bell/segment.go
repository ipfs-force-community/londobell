package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/dep"
	"github.com/dtynn/londobell/lib/fxex"
	"github.com/dtynn/londobell/racailum/segment"
)

var segmentCmd = &cli.Command{
	Name: "segment",
	Subcommands: []*cli.Command{
		segmentInitBoundaryCmd,
	},
}

var segmentInitBoundaryCmd = &cli.Command{
	Name: "init-boundary",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},

		&cli.StringFlag{
			Name: "hi-child",
		},

		&cli.StringFlag{
			Name: "lo-child",
		},

		dep.FlagMgoMetaMgrDSN,
	},

	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			MetaMgr common.MetaManager
			CS      common.ChainStore
		}

		app := dep.BellApp(
			cctx.Context,
			fxlog,
			&components,
			fxex.ProvideEx(
				dep.MgoMetaMgrDSN(cctx.String(dep.FlagMgoMetaMgrDSN.Name)),
			),
		)

		err := app.Start(cctx.Context)
		if err != nil {
			return err
		}

		defer app.Stop(cctx.Context)

		var hi, lo *common.LinkedTipSet
		if cctx.IsSet("hi-child") {
			tsk, err := parsetTipSetKey(cctx.String("hi-child"))
			if err != nil {
				return fmt.Errorf("hi-child key: %w", err)
			}

			hi, err = common.LoadLinkedTipSet(components.CS, tsk)
			if err != nil {
				return fmt.Errorf("load hi tipset: %w", err)
			}
		}

		if cctx.IsSet("lo-child") {
			tsk, err := parsetTipSetKey(cctx.String("lo-child"))
			if err != nil {
				return fmt.Errorf("lo-child key: %w", err)
			}

			lo, err = common.LoadLinkedTipSet(components.CS, tsk)
			if err != nil {
				return fmt.Errorf("load lo tipset: %w", err)
			}
		}

		if hi == nil && lo == nil {
			log.Warn("no boundary provided")
			return nil
		}

		err = segment.SetBoundary(cctx.Context, cctx.String("name"), components.MetaMgr, hi, lo)
		if err != nil {
			return fmt.Errorf("set boundary: %w", err)
		}

		log.Infow("boundary set", "seg", cctx.String("name"), "hi", hi.String(), "lo", lo.String())
		return nil
	},
}
