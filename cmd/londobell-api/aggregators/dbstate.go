package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

var dbstateCmd = &cli.Command{
	Name: "dbstate",
	Subcommands: []*cli.Command{
		archiveCmd,
		loadCmd,
		updateCmd, // 继续加载state
		deleteCmd,
		//copyCmd, // todo
	},
}

// todo: 归档完删除原formal的dbState

// copy dbState of formal to cold after cold copy content of formal
// todo: 新增冷库的startEpoch要等于上个冷库的finalHeight+1(裁剪), 获取最后高度作为finalHeight入库（新增） 并算dbstate
// todo: 补充表的不完整的数据 不用
// cold startEpoch从上个冷库的endEpoch开始，cold endEpoch总是等于finalHeight+1
// 确保目前的colds是正确的
// formal和cold的数据库是一致的
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
		&cli.IntFlag{
			Name: "limit",
		},
		&cli.IntFlag{
			Name: "interval",
		},
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
			Usage: "delete the original data of cold if force is true",
		},
	},
	Action: func(cctx *cli.Context) error {
		limit, interval := cctx.Int("limit"), cctx.Int("interval")
		if limit == 0 || interval == 0 {
			return fmt.Errorf("invalid limit %v or interval %v", limit, interval)
		}

		var components struct {
			DBStMgr multiquery.DataBaseStateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			multiquery.MultiQuery(context.TODO(), &components.DBStMgr),
			multiquery.InjectRepoPath(cctx),
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

		coldDB := multiquery.NewDB(coldURL, coldName)
		formalDB := multiquery.NewDB(formalURL, formalName)

		if formalDB.IsInvalidDB() || coldDB.IsInvalidDB() {
			log.Errorf("formal %v or cold %v is invalid db", formalDB, coldDB)
			return nil
		}

		cols, err := multiquery.GetCollectionsForDB(cctx.Context, formalDB)
		if err != nil {
			log.Errorf("get collections for DB %v failed: %v", formalDB, cols)
			return err
		}

		//formalState, found, err := components.DBStMgr.Stm.LoadDataBaseState(formalURL)
		formalState, found, err := components.DBStMgr.Seg.Find(cctx.Context, formalURL)
		if !found {
			log.Errorf("no url %v found in dbstate", formalURL)
			return nil
		}
		if err != nil {
			return err
		}

		err = CompleteDataBaseState(cctx.Context, &formalState, cols, limit, interval)
		if err != nil {
			return err
		}

		// todo: formal-url的dbstate就不用补了
		//_, found, err = components.DBStMgr.Stm.LoadDataBaseState(coldURL)
		_, found, err = components.DBStMgr.Seg.Find(cctx.Context, formalURL)
		if err != nil {
			log.Errorf("load dbState for %v failed: %v", coldURL, err)
			return err
		}

		if found {
			if !cctx.Bool("force") {
				log.Errorf("url %v found in dbstate, but force is not true", coldURL)
				return nil
			}
		}

		// todo: formalState.startEpoch 不会与colds范围重合

		// 覆盖cold
		_, err = components.DBStMgr.Seg.Update(cctx.Context, coldURL, formalState)
		//err := components.DBStMgr.Stm.SetDataBaseState(coldURL, formalState)
		if err != nil {
			return err
		}

		cfg := components.DBStMgr.GetCfg()
		exist := ColdsIsExists(coldDB, cfg.Colds)

		// 更新config
		if !exist {
			cfg.Formal = multiquery.DB{}
			validColds := make([]multiquery.DB, 0)
			for _, cold := range cfg.Colds {
				if cold.IsInvalidDB() {
					log.Warnw("invalid cold", "url", cold.Url(), "name", cold.Name())
				}

				validColds = append(validColds, cold)
			}

			cfg.Colds = append(validColds, coldDB)

			err = multiquery.WriteToConfig(multiquery.ConfigFilePath(components.DBStMgr.GetRepoPath()), cfg)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

// 除loadCmd外，其他接口访问都要停止进程？
var loadCmd = &cli.Command{
	Name: "load",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "RPCListen",
			Usage: "multiaddr of rpc",
		},
		&cli.BoolFlag{
			Name:  "local",
			Usage: "load locally if true, otherwise rpc call if false",
		},
	},
	Action: func(cctx *cli.Context) error {
		url := cctx.String("url")
		local := cctx.Bool("local")

		var (
			dbState multiquery.DataBaseState
			found   bool
		)
		if local {
			var components struct {
				DBStMgr multiquery.DataBaseStateManager
			}

			stopper, err := dix.New(
				cctx.Context,
				multiquery.MultiQuery(context.TODO(), &components.DBStMgr),
				multiquery.InjectRepoPath(cctx),
			)
			if err != nil {
				fmt.Println("stopper", err)
				return err
			}

			defer stopper(cctx.Context) // nolint: errcheck

			dbState, found, err = components.DBStMgr.Seg.Find(context.TODO(), url)
			//dbState, found, err = components.DBStMgr.Stm.LoadDataBaseState(url)
			if err != nil {
				return err
			}

			if !found {
				log.Warnf("url %v not exist", url)
				return nil
			}
		} else {
			api, _, err := GetAPIV0(cctx.Context, cctx.String("RPCListen"))
			if err != nil {
				return err
			}

			dbState, err = api.LoadDBState(url)
			if err != nil {
				return err
			}
		}

		log.Infof("dbState of url %v: %+v", url, dbState)

		return nil
	},
}

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
		&cli.StringFlag{
			Name:     "type",
			Usage:    "type: BlockMsgsCount, BlockMsgsByMethodNameMap, ActorMsgsByMethodNameMap, ActorMsgsCountMap, ActorTransfersCountMap, MinedMsgsMap, TransfersLargeAmountCount or all",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:  "apis",
			Usage: "ws://112.124.1.253:1234/rpc/v0",
		},
		&cli.Int64Flag{
			Name:     "end",
			Required: true,
		},
		&cli.IntFlag{
			Name: "limit",
		},
		&cli.IntFlag{
			Name: "interval",
		},
	},
	Action: func(cctx *cli.Context) error {
		limit, interval := cctx.Int("limit"), cctx.Int("interval")
		if limit == 0 || interval == 0 {
			return fmt.Errorf("invalid limit %v or interval %v", limit, interval)
		}

		fullnode.API = fullnode.NewAppropriateAPI(cctx.StringSlice("apis"))
		err := fullnode.API.Choose(context.TODO())
		if err != nil {
			return err
		}

		var components struct {
			DBStMgr multiquery.DataBaseStateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			multiquery.MultiQuery(context.TODO(), &components.DBStMgr),
			multiquery.InjectRepoPath(cctx),
		)
		if err != nil {
			fmt.Println("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		url := cctx.String("url")
		name := cctx.String("name")
		utype := cctx.String("type")
		endEpoch := cctx.Int64("end")

		err = updateBaseStateForType(context.TODO(), url, name, utype, abi.ChainEpoch(endEpoch), &components.DBStMgr, limit, interval)
		if err != nil {
			return err
		}

		log.Infof("update dbstate of %v for %v successfully", url, utype)
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
			Name:     "type",
			Usage:    "type: BlockMsgsCount, BlockMsgsByMethodNameMap, ActorMsgsByMethodNameMap, ActorMsgsCountMap, ActorTransfersCountMap, MinedMsgsMap, TransfersLargeAmountCount or all",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		var components struct {
			DBStMgr multiquery.DataBaseStateManager
		}

		stopper, err := dix.New(
			cctx.Context,
			multiquery.MultiQuery(context.TODO(), &components.DBStMgr),
			multiquery.InjectRepoPath(cctx),
		)
		if err != nil {
			fmt.Println("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		url := cctx.String("url")
		dtype := cctx.String("type")

		err = deleteDataBaseStateForType(url, dtype, &components.DBStMgr)
		if err != nil {
			return err
		}

		log.Infof("delete dbstate of %v for %v successfully", url, dtype)
		return nil
	},
}

// first delete, then reload
func updateBaseStateForType(ctx context.Context, url, name, utype string, endEpoch abi.ChainEpoch, dBStMgr *multiquery.DataBaseStateManager, limit, interval int) error {
	dbState, found, err := dBStMgr.Seg.Find(ctx, url)
	//dbState, found, err := dBStMgr.Stm.LoadDataBaseState(url)
	if err != nil {
		return err
	}

	if !found {
		log.Warnf("url %v not exist", url)
		return nil
	}

	cols, err := multiquery.GetCollectionsForDB(ctx, multiquery.NewDB(url, name))
	if err != nil {
		log.Errorf("get collections for url %v failed: %v", url, err)
		return err
	}

	switch utype {
	case "BlockMsgsCount":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshBlockMsgs(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		//return dBStMgr.Stm.SetDataBaseState(url, dbState)
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "BlockMsgsByMethodNameMap":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshBlockMsgsByMethodName(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "ActorMsgsByMethodNameMap":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshActorMsgsByMethodName(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "ActorMsgsCountMap":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshActorMsgs(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "ActorTransfersCountMap":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshActorTransferMsgs(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "MinedMsgsMap":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshMinedMsgsMaps(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "TransfersLargeAmountCount":
		dbState.EndEpoch = endEpoch
		if err := multiquery.RefreshTransfersForLargeAmount(ctx, &dbState, cols, limit, interval); err != nil {
			return err
		}
		_, err = dBStMgr.Seg.Update(ctx, url, dbState)
		return err
	case "all":
		for dbState.EndEpoch <= endEpoch {
			dbState.EndEpoch = dbState.EndEpoch + abi.ChainEpoch(limit*interval)
			if dbState.EndEpoch > endEpoch {
				dbState.EndEpoch = endEpoch
			}

			var ewg multierror.Group
			for i := range multiquery.Refreshes {
				i := i
				refresh := multiquery.Refreshes[i]
				ewg.Go(func() error {
					if err := refresh(ctx, &dbState, cols, limit, interval); err != nil {
						return err
					}

					return nil
				})
			}

			if err := ewg.Wait(); err != nil {
				log.Errorf("RefreshFormalDataBaseState failed: %v", err)
				return err
			}

			_, err = dBStMgr.Seg.Update(ctx, url, dbState)
			if err != nil {
				return err
			}
		}

		return nil

	default:
		return fmt.Errorf("invalid dtype: %v", utype)
	}
}

func deleteDataBaseStateForType(url, dtype string, dBStMgr *multiquery.DataBaseStateManager) error {
	dbState, found, err := dBStMgr.Seg.Find(context.TODO(), url)
	//dbState, found, err := dBStMgr.Stm.LoadDataBaseState(url)
	if err != nil {
		return err
	}

	if !found {
		log.Warnf("url %v not exist", url)
		return nil
	}

	switch dtype {
	case "BlockMsgsCount":
		dbState.BlockMsgsStates = multiquery.DefaultBlockMsgsStates(dbState.StartEpoch)
	case "BlockMsgsByMethodNameMap":
		dbState.BlockMsgsByMethodNameStates = multiquery.DefaultBlockMsgsByMethodNameStates(dbState.StartEpoch)
	case "ActorMsgsByMethodNameMap":
		dbState.ActorMsgsByMethodNameStates = multiquery.DefaultActorMsgsByMethodNameStates(dbState.StartEpoch)
		//return dBStMgr.Stm.SetDataBaseState(url, dbState)
	case "ActorMsgsCountMap":
		dbState.ActorMsgsCountStates = multiquery.DefaultActorMsgsCountStates(dbState.StartEpoch)
	case "ActorTransfersCountMap":
		dbState.ActorTransfersCountStates = multiquery.DefaultActorTransfersCountStates(dbState.StartEpoch)
	case "MinedMsgsMap":
		dbState.MinedMsgsStates = multiquery.DefaultMinedMsgsStates(dbState.StartEpoch)
	case "TransfersLargeAmountCount":
		dbState.TransfersLargeAmountStates = multiquery.DefaultTransfersLargeAmountStates(dbState.StartEpoch)
	case "all":
		//return dBStMgr.Stm.DeleteDataBaseState(url)
		dbState = *multiquery.DefaultDataBaseState(dbState.Formal, dbState.Tmp, dbState.StartEpoch, dbState.EndEpoch)
	default:
		return fmt.Errorf("invalid dtype: %v", dtype)
	}

	_, err = dBStMgr.Seg.Update(context.TODO(), url, dbState)
	return err
}
