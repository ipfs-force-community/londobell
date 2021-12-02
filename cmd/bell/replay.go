package main

import (
	"encoding/json"
	"fmt"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/replaytool"
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
			Usage:    "the messages need to replay",
		},
		&cli.StringFlag{
			Name:     "tipsetkey-cids-string",
			Aliases:  []string{"t"},
			Usage:    "the cids string of tipsetkey for a tipset to replay messages",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "output-filepath",
			Aliases: []string{"f"},
			Usage:   "the filepath to save replay results",
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
			return fmt.Errorf("unmarshall message err: %w", err)
		}

		//获取tipset
		var ts *types.TipSet
		tsk, err := parsetTipSetKey(cctx.String("tipsetkey-cids-string"))
		if err != nil {
			return fmt.Errorf("parse tsk err: %w", err)
		}

		ts, err = components.CS.LoadTipSet(tsk)
		if err != nil {
			return fmt.Errorf("failed to load tipset for tipsetkey %s: %w", tsk, err)
		}

		var filepath *string
		output := cctx.String("output-filepath")
		if output != "" {
			filepath = &output
		}

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
