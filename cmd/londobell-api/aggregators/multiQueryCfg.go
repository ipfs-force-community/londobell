package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

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
		rpath, err := multiquery.GetRepoPath(cctx)
		if err != nil {
			return err
		}

		cfgPath := multiquery.ConfigFilePath(rpath)

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

		cfg := multiquery.DefaultConfig()
		err = multiquery.WriteToConfig(cfgPath, cfg)
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

// formal 只更新最近finalheight
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
		&cli.StringFlag{
			Name:     "db-type",
			Required: true,
			Usage:    "three types: tmp, formal, cold",
		},
		//&cli.StringSliceFlag{
		//	Name:  "apis",
		//	Usage: "ws://112.124.1.253:1234/rpc/v0",
		//},
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
	},
	Action: func(cctx *cli.Context) error {
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
			multiquery.MultiQuery(context.TODO(), &components.DBStMgr),
			multiquery.InjectRepoPath(cctx),
		)
		if err != nil {
			fmt.Println("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		newURL := cctx.String("new-url")
		newName := cctx.String("new-name")
		force := cctx.Bool("force")

		newDB := multiquery.NewDB(newURL, newName)
		cfg := components.DBStMgr.GetCfg()

		formal, tmp := false, false
		dbType := cctx.String("db-type")
		switch dbType {
		case "tmp":
			cfg.Tmp = newDB
			tmp = true
		case "formal":
			cfg.Formal = newDB
			formal = true
		case "cold":
			colds := cfg.Colds

			validColds := make([]multiquery.DB, 0)
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

		if !tmp {
			// todo: 如果load存在？
			_, ok, err := components.DBStMgr.Stm.LoadDataBaseState(newDB.Url())
			if err != nil {
				log.Errorf("load dbState for newDB %v failed: %v", newDB, err)
			} else if ok && !force {
				log.Warnf("dbState of newDB %v exists, don't reload because of force be %v", newDB, force)
			} else if ok && force || !ok {
				log.Infof("set dbState for newDB %v, ok: %v, force: %v", newDB, ok, force)

				err = components.DBStMgr.FirstSetDataBaseState(cctx.Context, newDB, dbType, formal, tmp)
				if err != nil {
					return err
				}
			}
		}

		err = multiquery.WriteToConfig(multiquery.ConfigFilePath(components.DBStMgr.GetRepoPath()), cfg)
		if err != nil {
			return err
		}

		log.Infof("update config successfully, elapsed: %v", time.Now().Sub(start).String())
		return nil
	},
}
