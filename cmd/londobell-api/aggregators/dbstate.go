package main

import (
	"context"
	"fmt"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"

	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/dep"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

var dbstateCmd = &cli.Command{
	Name: "dbstate",
	Subcommands: []*cli.Command{
		archiveCmd,
		//loadCmd,
		updateCmd,
		deleteCmd,
	},
}

// archive formal to cold
var archiveCmd = &cli.Command{
	Name: "archive",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "formal-url",
		},
		&cli.StringFlag{
			Name: "formal-name",
		},
		&cli.StringFlag{
			Name: "cold-url",
		},
		&cli.StringFlag{
			Name: "cold-name",
		},
		&cli.Int64Flag{
			Name:  "interval",
			Usage: "interval of segment, 3*builtin.EpochsInHour default",
		},
	},
	Action: func(cctx *cli.Context) error {
		var components struct {
			DBStMgr multiquery.DataBaseStateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			dep.MultiQuery(context.TODO(), &components.DBStMgr),
			dep.InjectRepoPath(cctx),
		)
		if err != nil {
			fmt.Println("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		formalURL := cctx.String("formal-url")
		coldURL := cctx.String("cold-url")
		formalName := cctx.String("formal-name")
		coldName := cctx.String("cold-name")
		var interval int64
		if cctx.IsSet("interval") {
			interval = cctx.Int64("interval")
		} else {
			interval = smodel.DefaultInterval
		}

		coldDB := common2.NewDB(coldURL, coldName)
		formalDB := common2.NewDB(formalURL, formalName)

		if formalDB.IsInvalidDB() || coldDB.IsInvalidDB() {
			log.Errorf("formal %v or cold %v is invalid db", formalDB, coldDB)
			return fmt.Errorf("formal %v or cold %v is invalid db", formalDB, coldDB)
		}

		cols, err := multiquery.GetCollectionsForDB(cctx.Context, formalDB)
		if err != nil {
			log.Errorf("get collections for DB %v failed: %v", formalDB, cols)
			return err
		}

		// delete formal state & config
		err = components.DBStMgr.DeleteAllState(cctx.Context, formalDB)
		if err != nil {
			log.Errorf("delete all state for %v failed: %v", formalDB, err)
		}

		cfg := components.DBStMgr.GetCfg()
		cfg.Formal = common2.EmptyDB()
		components.DBStMgr.SetConfig(cfg)

		// load colds state & update config
		err = components.DBStMgr.FirstSetDataBaseState(cctx.Context, coldDB, smodel.Cold, interval)
		if err != nil {
			return err
		}

		cfg = components.DBStMgr.GetCfg()
		exist := ColdsIsExists(coldDB, cfg.Colds)
		if !exist {
			cfg.Colds = append(cfg.Colds, coldDB)
		}

		repoPath, err := dep.GetRepoPath(cctx)
		if err != nil {
			return err
		}

		err = common2.WriteToConfig(dep.ConfigFilePath(repoPath), cfg)
		if err != nil {
			return err
		}

		return nil
	},
}

//// 除loadCmd外，其他接口访问都要停止进程？
//var loadCmd = &cli.Command{
//	Name: "load",
//	Flags: []cli.Flag{
//		&cli.StringFlag{
//			Name:     "url",
//			Required: true,
//		},
//		&cli.StringFlag{
//			Name:  "RPCListen",
//			Usage: "multiaddr of rpc",
//		},
//		&cli.BoolFlag{
//			Name:  "local",
//			Usage: "load locally if true, otherwise rpc call if false",
//		},
//	},
//	Action: func(cctx *cli.Context) error {
//		url := cctx.String("url")
//		local := cctx.Bool("local")
//
//		var (
//			dbState multiquery.DataBaseState
//			found   bool
//		)
//		if local {
//			var components struct {
//				DBStMgr multiquery.DataBaseStateManager
//			}
//
//			stopper, err := dix.New(
//				cctx.Context,
//				multiquery.MultiQuery(context.TODO(), &components.DBStMgr),
//				multiquery.InjectRepoPath(cctx),
//			)
//			if err != nil {
//				fmt.Println("stopper", err)
//				return err
//			}
//
//			defer stopper(cctx.Context) // nolint: errcheck
//
//			dbState, found, err = components.DBStMgr.Stm.LoadDataBaseState(url)
//			if err != nil {
//				return err
//			}
//
//			if !found {
//				log.Warnf("url %v not exist", url)
//				return nil
//			}
//		} else {
//			api, _, err := GetAPIV0(cctx.Context, cctx.String("RPCListen"))
//			if err != nil {
//				return err
//			}
//
//			dbState, err = api.LoadDBState(url)
//			if err != nil {
//				return err
//			}
//		}
//
//		log.Infof("dbState of url %v: %+v", url, dbState)
//
//		return nil
//	},
//}

// updateCmd reload dbState from StartEpoch to EndEpoch
var updateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "utype",
			Usage:    "type: BlockStates(0), BlockMethodStates(1), BlockHeaderMethodStates(2), ActorStates(3), ActorMethodStates(4), ActorTransferStates(5), ActorEventStates(6), MinedStates(7), LargeAmountTransferStates(8), DealState(9), DealActorStates(10), TipSetStates(11), AllStates(12)",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		var components struct {
			DBStMgr multiquery.DataBaseStateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			dep.MultiQuery(context.TODO(), &components.DBStMgr),
			dep.InjectRepoPath(cctx),
		)

		if err != nil {
			fmt.Println("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		url := cctx.String("url")
		name := cctx.String("name")
		utype := cctx.Int("utype")

		db := common2.NewDB(url, name)
		if db.IsInvalidDB() {
			return fmt.Errorf("db %v is invalid", db)
		}

		err = updateBaseStateForType(cctx.Context, &components.DBStMgr, db, multiquery.Ptype(utype))
		if err != nil {
			return err
		}

		log.Infof("update db state of %v for %v successfully", utype, db)
		return nil
	},
}

var deleteCmd = &cli.Command{
	Name: "delete",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "dtype",
			Usage:    "type: BlockStates(0), BlockMethodStates(1), BlockHeaderMethodStates(2), ActorStates(3), ActorMethodStates(4), ActorTransferStates(5), ActorEventStates(6), MinedStates(7), LargeAmountTransferStates(8), DealState(9), DealActorStates(10), TipSetStates(11), AllStates(12)",
			Required: true,
		},
		&cli.IntFlag{
			Name:  "db-type",
			Usage: "three types: 0(tmp), 1(formal), 2(cold)",
		},
	},
	Action: func(cctx *cli.Context) error {
		var components struct {
			DBStMgr multiquery.DataBaseStateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			dep.MultiQuery(context.TODO(), &components.DBStMgr),
			dep.InjectRepoPath(cctx),
		)

		if err != nil {
			fmt.Println("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		url := cctx.String("url")
		name := cctx.String("name")
		dtype := cctx.Int("dtype")
		dbtype := smodel.DType(cctx.Int("db-type"))

		db := common2.NewDB(url, name)
		if db.IsInvalidDB() {
			return fmt.Errorf("db %v is invalid", db)
		}

		err = deleteDataBaseStateForType(cctx.Context, &components.DBStMgr, db, multiquery.Ptype(dtype))
		if err != nil {
			return err
		}

		if multiquery.Ptype(dtype) == multiquery.AllStates {
			if !cctx.IsSet("db-type") {
				log.Errorf("delete all states must set db-type")
				return nil
			}

			cfg := components.DBStMgr.GetCfg()
			switch dbtype {
			case smodel.Tmp:
				cfg.Tmp = common2.EmptyDB()
			case smodel.Formal:
				cfg.Formal = common2.EmptyDB()
			case smodel.Cold:
				newColds := make([]common2.DB, 0)
				for _, cold := range cfg.Colds {
					if db.Equals(cold) {
						continue
					}
					newColds = append(newColds, cold)
				}
				cfg.Colds = newColds
			default:
				log.Errorf("invalid db type %v", dbtype)
				return nil
			}

			repoPath, err := dep.GetRepoPath(cctx)
			if err != nil {
				return err
			}

			err = common2.WriteToConfig(dep.ConfigFilePath(repoPath), cfg)
			if err != nil {
				return err
			}

			log.Infof("update config %v successfully", dep.ConfigFilePath(repoPath))
		}

		log.Infof("delete db state of %v for %v successfully", dtype, db)
		return nil
	},
}

func updateBaseStateForType(ctx context.Context, dBStMgr *multiquery.DataBaseStateManager, db common2.DB, dtype multiquery.Ptype) error {
	state, found, err := dBStMgr.GetState(ctx, db.Url())
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("db %v not found", db)
	}
	addupState := segment.NewAddUpState(*state)

	cols, err := multiquery.GetCollectionsForDB(ctx, db)
	if err != nil {
		return err
	}

	finalHeight, err := multiquery.GetFinalHeight(ctx, cols)
	if err != nil {
		return err
	}

	nextEndEpoch := int64(finalHeight + 1)

	switch dtype {
	case multiquery.BlockStates:
		if err := dBStMgr.UpdateBlockState(ctx, nextEndEpoch, state, addupState, cols); err != nil {
			return err
		}
	case multiquery.BlockMethodStates:
		if err := dBStMgr.UpdateBlockMethodState(ctx, nextEndEpoch, state, addupState, cols); err != nil {
			return err
		}
	// 其他暂时不处理
	case multiquery.BlockHeaderMethodStates:
	case multiquery.ActorStates:
	case multiquery.ActorMethodStates:
	case multiquery.ActorTransferStates:
	case multiquery.ActorEventStates:
	case multiquery.MinedStates:
	case multiquery.LargeAmountTransferStates:
	case multiquery.DealState:
	case multiquery.DealActorStates:
	case multiquery.TipSetStates:
	case multiquery.AllStates:
		if err := dBStMgr.UpdateAllState(ctx, nextEndEpoch, state, addupState, cols); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid dtype: %v", dtype)
	}

	return nil
}

func deleteDataBaseStateForType(ctx context.Context, dBStMgr *multiquery.DataBaseStateManager, db common2.DB, dtype multiquery.Ptype) error {
	switch dtype {
	case multiquery.BlockStates:
		if err := dBStMgr.DeleteBlockState(ctx, db); err != nil {
			return err
		}
	case multiquery.BlockMethodStates:
		if err := dBStMgr.DeleteBlockMethodState(ctx, db); err != nil {
			return err
		}
	case multiquery.ActorStates:
		if err := dBStMgr.DeleteActorState(ctx, db); err != nil {
			return err
		}
	case multiquery.ActorMethodStates:
		if err := dBStMgr.DeleteActorMethodState(ctx, db); err != nil {
			return err
		}
	// 其他暂时不处理
	case multiquery.BlockHeaderMethodStates:
	case multiquery.ActorTransferStates:
	case multiquery.ActorEventStates:
	case multiquery.MinedStates:
	case multiquery.LargeAmountTransferStates:
	case multiquery.DealState:
	case multiquery.DealActorStates:
	case multiquery.TipSetStates:
	case multiquery.AllStates:
		if err := dBStMgr.DeleteAllState(ctx, db); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid dtype: %v", dtype)
	}

	return nil
}
