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
			Usage: "ws://112.124.1.253:1234/rpc/v0",
		},
		&cli.StringFlag{
			Name:  "RPCListen",
			Usage: "multiaddr of rpc",
		},
		&cli.IntFlag{
			Name: "limit",
		},
		&cli.IntFlag{
			Name: "interval",
		},
	},
	Action: func(cctx *cli.Context) error {
		err := server.Run(cctx, false, cctx.Int("limit"), cctx.Int("interval"))
		if err != nil {
			return err
		}

		return nil
	},
}
