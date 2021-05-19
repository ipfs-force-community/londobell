package dep

import "github.com/urfave/cli/v2"

// common flags
var (
	FullNodeAPIFlag = &cli.StringFlag{
		Name: "api-url",
	}
)
