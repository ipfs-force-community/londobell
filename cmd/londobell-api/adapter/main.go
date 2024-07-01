package main

import (
	"os"

	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("lotus-api-adapter")

func main() {
	lotuslog.SetupLogLevels()
	logging.SetLogLevel("rpc", "FATAL")

	mcodec.Setup()
	app := &cli.App{
		Name:  "lotus-api-adapter",
		Usage: "chain data",
		Commands: []*cli.Command{
			daemonCmd,
		},
		Version: string(build.NodeUserVersion()),
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
