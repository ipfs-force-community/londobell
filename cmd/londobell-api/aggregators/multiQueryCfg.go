package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/dep"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// londobell dsn存在config里
var multiQueryCfgCmd = &cli.Command{
	Name: "multiquery-cfg",
	Subcommands: []*cli.Command{
		cfgInitCmd,
		cfgUpdateCmd,
		// 删除cold直接改配置文件即可
	},
}

var cfgInitCmd = &cli.Command{
	Name:  "init",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		rpath, err := dep.GetRepoPath(cctx)
		if err != nil {
			return err
		}

		cfgPath := dep.ConfigFilePath(rpath)

		_, err = os.Stat(cfgPath)
		if err == nil {
			log.Warn("config file already exists")
			return nil
		}

		log.Infof("init config: %s", cfgPath)

		if !os.IsNotExist(err) {
			return fmt.Errorf("fs error: %w", err)
		}

		err = os.MkdirAll(filepath.Dir(cfgPath), 0755)
		if err != nil {
			return fmt.Errorf("MkdirAll for %s: %w", cfgPath, err)
		}

		cfg := common.DefaultConfig()
		err = common.WriteToConfig(cfgPath, cfg)
		if err != nil {
			return err
		}

		log.Infof("init done, cfg: %v", cfg)

		return nil
	},
}

// Do not modify the config file directly! Use cfgUpdateCmd.
// 修改tmp可以直接改配置文件
// 第一次colds使用cfgUpdateCmd落盘，后面由老formal过渡到cold用archiveCmd

// todo: 第一次添加或更新cold、formal
var cfgUpdateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "new-url",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "new-name",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "db-type",
			Required: true,
			Usage:    "three types: 0(tmp), 1(formal), 2(cold)",
		},
		&cli.StringFlag{
			Name:     "nodeconfig",
			Usage:    "The location of the node configuration, eg: ./config.json(api: token)",
			Required: true,
		},

		//&cli.BoolFlag{
		//	Name:  "reverse",
		//	Value: false,
		//	Usage: "add cold in reverse order if reverse is true, otherwise in sequential order. false in default",
		//},
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
			Usage: "reload even if dbState exists if force is true, otherwise return. true in default",
		},
		&cli.Int64Flag{
			Name:  "interval",
			Usage: "interval of segment",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.TODO()
		start := time.Now()

		if err := util.ParseNodes(cctx.String("nodeconfig")); err != nil {
			return err
		}

		fullnode.API = fullnode.NewAppropriateAPI(util.Nodes)
		err := fullnode.API.Choose(context.TODO())
		if err != nil {
			return err
		}

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

		newURL := cctx.String("new-url")
		newName := cctx.String("new-name")
		force := cctx.Bool("force")
		var interval int64
		if cctx.IsSet("interval") {
			interval = cctx.Int64("interval")
		} else {
			interval = smodel.DefaultInterval
		}

		newDB := common.NewDB(newURL, newName)
		cfg := components.DBStMgr.GetCfg()

		dbType := smodel.DType(cctx.Int("db-type"))
		switch dbType {
		case smodel.Tmp:
			cfg.Tmp = newDB
		case smodel.Formal:
			cfg.Formal = newDB
		case smodel.Cold:
			colds := cfg.Colds

			validColds := make([]common.DB, 0)
			for _, cold := range colds {
				if cold.IsInvalidDB() {
					log.Warnw("invalid cold", "url", cold.Url(), "name", cold.Name())
					continue
				}

				if cold.Equals(newDB) {
					continue
				}

				validColds = append(validColds, cold)
			}

			exist := ColdsIsExists(newDB, colds)
			if exist {
				if !force {
					// todo: 手动删除
					log.Warnw("db exists in colds", "new-url", newURL, "new-name", newName, "db-type", dbType, "colds", colds)
					return nil
				}

				// delete exist newDB
				components.DBStMgr.ReplaceColdsCfg(validColds)
			}

			cfg.Colds = append(validColds, newDB)
		default:
			log.Errorf("invalid db-type, must be tmp, formal or colds")
			return nil
		}

		if dbType == smodel.Formal || dbType == smodel.Cold {
			// todo: 如果load存在？
			_, ok, err := components.DBStMgr.GetState(ctx, newDB.Url())
			if err != nil {
				return fmt.Errorf("load dbState for newDB %v failed: %v", newDB, err)
			}
			if ok && !force {
				log.Warnf("dbState of newDB %v exists, don't reload because of force be %v", newDB, force)
				return nil
			}

			if ok && force || !ok {
				log.Infof("set dbState for newDB %v, ok: %v, force: %v", newDB, ok, force)

				err = components.DBStMgr.FirstSetDataBaseState(ctx, newDB, dbType, interval)
				if err != nil {
					return err
				}
			}
		}
		repoPath, err := dep.GetRepoPath(cctx)
		if err != nil {
			return err
		}

		err = common.WriteToConfig(dep.ConfigFilePath(repoPath), cfg)
		if err != nil {
			return err
		}

		log.Infof("update config successfully, elapsed: %v", time.Now().Sub(start).String())
		return nil
	},
}
