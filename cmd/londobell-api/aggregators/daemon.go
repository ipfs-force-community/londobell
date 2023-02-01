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
		&cli.StringSliceFlag{
			Name:  "apis",
			Usage: "ws://127.0.0.1:1234/rpc/v0",
		},

		// todo: dbs of config.json
	},
	Action: func(cctx *cli.Context) error {
		err := server.Run(cctx, false)
		if err != nil {
			return err
		}

		return nil
	},
}
