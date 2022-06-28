package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/controller"
)

var log = logging.Logger("data")

func main() {
	app := &cli.App{
		Name:  "lotus-api-adapter",
		Usage: "chain data",
		Commands: []*cli.Command{
			daemonCmd,
			sectorPowerCmd,
		},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}

var sectorPowerCmd = &cli.Command{
	Name:  "sector-power",
	Usage: "get sector power for miner",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name: "epoch",
		},
		&cli.StringFlag{
			Name: "miner",
		},
		&cli.Uint64Flag{
			Name: "sector",
		},
	},
	Action: func(cctx *cli.Context) error {
		err := controller.GetSectorPower(cctx)
		if err != nil {
			return err
		}

		return nil
	},
}
