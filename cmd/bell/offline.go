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
	Usage: "offline extractor data,support custom start,end epoch",
}

var extractorCmd = &cli.Command{
	Name:  "extractor",
	Usage: "use local chain storage to extractor data",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start-height",
			Required: true,
			Usage:    "start height of start epoch (not inclusive)",
		},
		&cli.StringFlag{
			Name:     "end-key",
			Required: true,
			Usage:    "tipSetKey of end epoch, Separated by ',' (not inclusive)",
		},
		&cli.BoolFlag{
			Name:  "local",
			Value: true,
			Usage: "local or remote",
		},
		&cli.BoolFlag{
			Name:  "writableOffline",
			Value: false,
			Usage: "writable or readonly for offline extract",
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
			dep.WalkRaCalium(cctx, fxlog, &components),
			dep.InjectRepoPath(cctx),
			dep.InjectWritableOffline(cctx),
			dix.Override(new(dtypes.ShutdownChan), shutdownCh),
		)
		if err != nil {
			return err
		}
		ts, err := parsetTipSetKey(cctx.String("end-key"))
		if err != nil {
			return err
		}
		return components.Ra.WalkExtract(ctx, ts, abi.ChainEpoch(cctx.Int64("start-height")))
	},
}
