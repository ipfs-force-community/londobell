package main

import (
	"errors"
	"fmt"
	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/dep"
	"github.com/dtynn/londobell/tool/replay"
	"github.com/dtynn/londobell/tool/replay/util"
	"github.com/filecoin-project/lotus/chain/beacon"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"os"
)
func main() {
	var (
		log = logging.Logger("replay")
	)

	app := &cli.App{
		Name:                 "replay",
		Usage:                "replay message tool",
		EnableBashCompletion: true,
		//Version:
		Flags: []cli.Flag{
			//参数为空时？
			&cli.StringFlag{
				Name:    "tipsetkey-cid-string",
				Aliases: []string{"t"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "msgcids-string",
				Aliases: []string{"m"},
				Required: true,
			},
		},

		Action: func(cctx *cli.Context) error {
			var components struct {
				fx.In
				CS common.ChainStore
				SM common.StateManager
			}

			app, err := buildApp(&components)
			if err != nil {
				return err
			}
			//fmt.Println(components)

			err = app.Start(cctx.Context)
			if err != nil {
				return err
			}

			defer app.Stop(cctx.Context)

			//获取tipset
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

			var msgcids []cid.Cid
			if cctx.String("msgcids-string") == "" {
				return errors.New("param msgcids-string is null")
			} else {
				msgcids, err = util.ParseStringToCidArray(cctx.String("msgcids-string"))
				if err != nil {
					return fmt.Errorf("parse msgs err: %w", err)
				}
			}

			//基于tipset和消息cids重放
			invocResult, err := replay.Replay(cctx.Context, components.SM, ts, msgcids)
			if err != nil {
				return fmt.Errorf("replay err: %w", err)
			}

			util.WriteTofile(invocResult)

			return nil
		},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorw("cli error: %s", err)
		os.Exit(1)
	}
}


func buildApp(target interface{}) (*fx.App, error) {
	opts := make([]fx.Option, 0)
	opts = append(opts, fx.Provide(
		journal.NilJournal,
		func() store.WeightFunc {
			return filcns.Weight
		},
		modules.ChainStore,

		filcns.NewTipSetExecutor,
		func() vm.SyscallBuilder {
			return vm.Syscalls(ffiwrapper.ProofVerifier)
		},
		filcns.DefaultUpgradeSchedule,
		func(cs *store.ChainStore, dc dtypes.DrandSchedule) beacon.Schedule {
			rbp := modules.RandomBeaconParams{
				Cs:          cs,
				DrandConfig: dc,
			}
			b, err := modules.RandomSchedule(rbp, dtypes.AfterGenesisSet{})
			if err != nil {
				panic(fmt.Errorf("construct random schedule failed: %w", err))
			}
			return b
		},
		dep.ChainIOBlockstore, //
		dep.InMemMetadataDS,   //
		stmgr.NewStateManager,
	))

	if target != nil {
		opts = append(opts, fx.Populate(target))
	}
	return fx.New(opts...), nil
}
