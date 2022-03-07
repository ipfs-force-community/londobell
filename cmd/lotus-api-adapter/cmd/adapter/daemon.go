package main

import (
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/server"
)

var daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start api for chain data",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "port",
			Usage: "1234",
		},
	},
	Action: func(cctx *cli.Context) error {
		err := server.Run(cctx)
		if err != nil {
			return err
		}

		return nil
	},
}
