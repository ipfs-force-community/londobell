package main

import (
	"os"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/dep"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"

	"github.com/filecoin-project/lotus/lib/lotuslog"
)

var log = logging.Logger("data")

func main() {
	lotuslog.SetupLogLevels()
	app := &cli.App{
		Name:  "londobell-api-aggregators",
		Usage: "api for londobell-aggregators",
		Commands: []*cli.Command{
			multiQueryCfgCmd,
			segmentCmd,
			dbstateCmd,
			daemonCmd,
		},
		Flags: []cli.Flag{
			dep.RepoFlag,
		},
		Version: build.UserVersion(),
	}

	app.Setup()
	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
