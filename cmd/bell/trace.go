package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var traceCmd = &cli.Command{
	Name: "trace",
	Subcommands: []*cli.Command{
		setCmd,
		showCmd,
	},
}
var showCmd = &cli.Command{
	Name: "show",
	Action: func(cctx *cli.Context) error {
		api, _, err := GetAPIV0(cctx)
		if err != nil {
			return err
		}
		rate, err := api.GetSampleRate(cctx.Context)
		if err != nil {
			return fmt.Errorf("get sample rate: %v", err)
		}
		log.Infof("cur sample rate: %v", rate)
		return nil
	},
}

var setCmd = &cli.Command{
	Name: "set",
	Flags: []cli.Flag{
		&cli.Float64Flag{
			Name:     "rate",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		api, _, err := GetAPIV0(cctx)
		if err != nil {
			return err
		}
		rate := cctx.Float64("rate")
		old, err := api.SetSampleRate(cctx.Context, rate)
		if err != nil {
			return err
		}
		log.Infof("set trace rate, cur: %v, old: %v", rate, old)
		return nil
	},
}
