package main

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/api"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/replaytool"

	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

var replayCmd = &cli.Command{
	Name: "replay",
	Subcommands: []*cli.Command{
		replayRunCmd,
	},
}

var replayRunCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "message",
			FilePath: "./example/message.txt",
		},
		&cli.StringFlag{
			Name:    "tipsetkey-cid-string",
			Aliases: []string{"t"},
		},
		&cli.StringFlag{
			Name:    "filepath",
			Aliases: []string{"f"},
		},
	},

	Action: func(cctx *cli.Context) error {
		var components struct {
			fx.In
			CS common.ChainStore
			SM common.StateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			dep.Bell(cctx.Context, fxlog, &components),
			dep.InjectFullNode(cctx),
		)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		//加载message
		content := cctx.String("message")
		var msglist []types.Message

		err = json.Unmarshal([]byte(content), &msglist)
		if err != nil {
			return fmt.Errorf("unmarshall message err: %s", err)
		}

		//获取tipset
		var ts *types.TipSet
		if cctx.String("tipsetkey-cid-string") == "" {
			return fmt.Errorf("param tipsetkey-cid-string is null")
		}

		tsk, err := parsetTipSetKey(cctx.String("tipsetkey-cid-string"))
		if err != nil {
			return fmt.Errorf("parse tsk err: %w", err)
		}

		ts, err = components.CS.LoadTipSet(tsk)
		if err != nil {
			return fmt.Errorf("failed to load tipset: %s: %s", tsk, err)
		}

		filepath := cctx.String("filepath")

		//基于tipset和消息cids重放
		var result []*api.InvocResult
		result, err = replaytool.Replay(cctx.Context, components.SM, ts, msglist, filepath)
		if err != nil {
			return fmt.Errorf("replay err: %w", err)
		}

		log.Infow("print replay message result", "result", result)

		return nil
	},
}
