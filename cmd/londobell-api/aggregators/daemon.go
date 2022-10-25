package main

import (
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/server"
)

var daemonCmd = &cli.Command{
	Name: "daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "port",
			Usage: "2345",
		},
	},
	Action: func(cctx *cli.Context) error {
		err := server.Run(cctx, false)
		if err != nil {
			return err
		}

		return nil
	},
}
