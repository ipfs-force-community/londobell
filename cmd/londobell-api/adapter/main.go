package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/lib/lotuslog"
)

var log = logging.Logger("data")

func main() {
	lotuslog.SetupLogLevels()
	app := &cli.App{
		Name:  "lotus-api-adapter",
		Usage: "chain data",
		Commands: []*cli.Command{
			daemonCmd,
		},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
