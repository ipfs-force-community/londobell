package dep

import "github.com/urfave/cli/v2"

// common flags
var (
	FlagMgoMetaMgrDSN = &cli.StringFlag{
		Name:     "mgo-metamgr",
		Required: true,
	}
)
