package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("data")

func main() {
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
