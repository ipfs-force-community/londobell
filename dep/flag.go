package dep

import "github.com/urfave/cli/v2"

// common flags
var (
	FlagMgoBstoreDSN = &cli.StringFlag{
		Name:     "mgo-bstore",
		Required: true,
	}

	FlagMgoMetaDSDSN = &cli.StringFlag{
		Name:     "mgo-metads",
		Required: true,
	}

	FlagMgoMetaMgrDSN = &cli.StringFlag{
		Name:     "mgo-metamgr",
		Required: true,
	}
)
