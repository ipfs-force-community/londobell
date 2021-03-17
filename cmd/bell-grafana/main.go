package main

import (
	_ "net/http/pprof"
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/lib/lotuslog"

	"github.com/dtynn/londobell/lib/mgoutil/mcodec"
)

var log = logging.Logger("bell-grafana")

func main() {
	lotuslog.SetupLogLevels()

	mcodec.Setup()

	app := &cli.App{
		Name:                 "bell-grafana",
		Usage:                "grafana data source service of londobell",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			runCmd,
		},
		Version: build.CurrentCommit,
		Flags:   []cli.Flag{},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
