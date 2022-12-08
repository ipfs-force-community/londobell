package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"

	"github.com/filecoin-project/lotus/lib/lotuslog"
)

var log = logging.Logger("lotus-api-adapter")

func main() {
	lotuslog.SetupLogLevels()
	logging.SetLogLevel("rpc", "FATAL")
	app := &cli.App{
		Name:  "lotus-api-adapter",
		Usage: "chain data",
		Commands: []*cli.Command{
			daemonCmd,
		},
		Version: build.UserVersion(),
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
