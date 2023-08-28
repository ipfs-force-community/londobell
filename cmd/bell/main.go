package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/lib/lotuslog"

	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"
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
			cfgCmd,
			segmentCmd,
			daemonCmd,
			traceCmd,
			compareCmd,
			checkCmd,
			offlineCmd,
			fixActorMessageCmd,
			extractAllSectorsCmd,
		},
		Version: build.UserVersion(),
		Flags: []cli.Flag{
			dep.RepoFlag,
			dep.FullNodeAPIFlag,
			dep.TokenFlag,
			dep.OfflineChainStorageRepoFlag,
		},
	}

	app.SliceFlagSeparator = ";"
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
