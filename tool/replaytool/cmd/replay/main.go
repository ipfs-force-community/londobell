package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/dep"
	"github.com/dtynn/londobell/lib/fxex"
	"github.com/dtynn/londobell/tool/replaytool/util"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"os"
)

func main() {
	app := &cli.App{
		Name:                 "replay",
		Usage:                "replay tool",
		EnableBashCompletion: true,
		//Version:
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "message",
				FilePath: "./example/message.txt",
			},
			dep.FullNodeAPIFlag,

			//参数为空时？
			&cli.StringFlag{
				Name:    "tipsetkey-cid-string",
				Aliases: []string{"t"},
				//Required: true,
			},
		},

		Action: func(cctx *cli.Context) error {
			//加载message
			content := cctx.String("message")
			fmt.Println(content)
			var msglist []types.Message
			json.Unmarshal([]byte(content), &msglist)

			full, closer, err := getFullNode(cctx)
			if err != nil {
				log.Errorf("getFullNode err: %s", err)
				return err
			}
			defer closer()

			var components struct {
				fx.In
				CS common.ChainStore
				SM common.StateManager
			}

			app, err := buildApp(cctx, full, &components)
			if err != nil {
				return err
			}
			//fmt.Println(components)
			err = app.Start(cctx.Context)
			if err != nil {
				return err
			}

			defer app.Stop(cctx.Context)

			//获取tipset  tipset为空时？
			var ts *types.TipSet
			if cctx.String("tipsetkey-cid-string") == "" {
				return errors.New("param tipsetkey-cid-string is null")

			} else {
				tsk, err := util.ParseTipSetKey(cctx.String("tipsetkey-cid-string"))
				//types.EmptyTSK时重放吗？？
				if err != nil {
					return fmt.Errorf("parse tsk err: %w", err)
				}

				ts, err = components.CS.LoadTipSet(tsk)
				if err != nil {
					return fmt.Errorf("failed to load tipset: %s: %s", tsk, err)
				}
			}

			//基于tipset和消息cids重放
			replayResult, err := replay(cctx.Context, components.SM, ts, msglist)
			if err != nil {
				return fmt.Errorf("replay err: %w", err)
			}

			util.WriteTofile(replayResult)

			return nil
		},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}

func buildApp(cctx *cli.Context, full v0api.FullNode, target interface{}) (*fx.App, error) {
	return dep.BellApp(
		cctx.Context,
		fxlog,
		target,
		fxex.ProvideEx(
			fxex.As(full, new(v0api.FullNode)),
		),
	), nil
}
