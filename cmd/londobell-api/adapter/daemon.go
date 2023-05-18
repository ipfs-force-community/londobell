package main

import (
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/server"
	"github.com/ipfs-force-community/londobell/dep"
)

var daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start api for chain data",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "port",
			Usage: "1234",
		},
		&cli.StringSliceFlag{
			Name:  "apis",
			Usage: "ws://127.0.0.1:1234/rpc/v0",
		},
		dep.RepoFlag,
	},
	Action: func(cctx *cli.Context) error {
		err := server.Run(cctx, true)
		if err != nil {
			return err
		}

		return nil
	},
}
