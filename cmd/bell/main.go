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

func main() {
	lotuslog.SetupLogLevels()

	// TODO: see if we should learn more about vm execution from logs
	logging.SetLogLevel("vm", "ERROR")

	mcodec.Setup()

	app := &cli.App{
		Name:                 "bell",
		Usage:                "chain info manager of Filecoin",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			raCmd,
			segmentCmd,
		},
		Version: build.CurrentCommit,
		Flags:   []cli.Flag{},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %v", err)
		os.Exit(1)
	}
}
