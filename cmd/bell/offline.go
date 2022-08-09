package main

import (
	"context"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/racailum"
)

var offlineCmd = &cli.Command{
	Name: "offline",
	Subcommands: []*cli.Command{
		extractorCmd,
	},
}

var extractorCmd = &cli.Command{
	Name:  "extractor",
	Usage: "use local chain storage to extractor data",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start-height",
			Required: true,
			Usage:    "start height of start epoch",
		},
		&cli.StringFlag{
			Name:     "end-key",
			Required: true,
			Usage:    "tipSetKey of end epoch, Separated by ',' ",
		},
		&cli.BoolFlag{
			Name:  "offline",
			Value: true,
			Usage: "online when false",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		shutdownCh := make(chan struct{})
		var components struct {
			fx.In
			Ra *racailum.RaCailum
		}
		_, err := dix.New(ctx,
			dix.ApplyIf(func(s *dix.Settings) bool {
				return cctx.Bool("offline")
			}, dep.InjectChainRepo(cctx)),
			dix.ApplyIf(func(s *dix.Settings) bool {
				return !cctx.Bool("offline")
			}, dep.InjectFullNode(cctx)),
			dix.ApplyIf(func(s *dix.Settings) bool {
				return cctx.Bool("offline")
			}, dep.OfflineRaCalium(ctx, fxlog, &components)),
			dix.ApplyIf(func(s *dix.Settings) bool {
				return !cctx.Bool("offline")
			}, dep.OnlineRaCalium(ctx, fxlog, &components)),
			dep.InjectRepoPath(cctx),
			dix.Override(new(dtypes.ShutdownChan), shutdownCh),
		)
		if err != nil {
			return err
		}
		ts, err := parsetTipSetKey(cctx.String("end-key"))
		if err != nil {
			return err
		}
		return components.Ra.OfflineExtract(ctx, ts, abi.ChainEpoch(cctx.Int64("start-height")))
	},
}
