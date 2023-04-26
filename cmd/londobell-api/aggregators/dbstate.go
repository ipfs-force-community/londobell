package main

import (
	"context"
	"fmt"

	"github.com/dtynn/dix"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/urfave/cli/v2"
)

var dbstateCmd = &cli.Command{
	Name: "dbstate",
	Subcommands: []*cli.Command{
		archiveCmd,
		loadCmd,
		deleteCmd,
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
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
			Usage: "delete the original data of cold if force is true",
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

		formalState, found, err := components.DBStMgr.Stm.LoadDataBaseState(formalURL)
		if !found {
			log.Errorf("no url %v found in dbstate", formalURL)
			return nil
		}
		if err != nil {
			return err
		}

		err = CompleteDataBaseState(cctx.Context, &formalState, cols)
		if err != nil {
			return err
		}

		// todo: formal-url的dbstate就不用补了
		_, found, err = components.DBStMgr.Stm.LoadDataBaseState(coldURL)
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
		if err := components.DBStMgr.Stm.SetDataBaseState(coldURL, formalState); err != nil {
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
	},
	Action: func(cctx *cli.Context) error {
		api, _, err := GetAPIV0(cctx.Context, cctx.String("RPCListen"))
		if err != nil {
			return err
		}

		dbState, err := api.LoadDBState(cctx.String("url"))
		if err != nil {
			return err
		}

		log.Infof("dbState of url %v: %+v", cctx.String("url"), dbState)
		return nil
	},
}

var deleteCmd = &cli.Command{
	Name: "delete",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "url",
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
		err = components.DBStMgr.Stm.DeleteDataBaseState(url)
		if err != nil {
			return err
		}

		log.Infof("delete dbstate of %v successfully", url)
		return nil
	},
}
