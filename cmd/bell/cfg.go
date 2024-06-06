package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/node/config"

	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/racailum"
)

var cfgCmd = &cli.Command{
	Name:  "cfg",
	Usage: "config for bell",
	Subcommands: []*cli.Command{
		cfgInitCmd,
	},
}

var cfgInitCmd = &cli.Command{
	Name:  "init",
	Flags: []cli.Flag{},
	Usage: "initialize config for bell",
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

		cfg := racailum.DefaultConfig()
		content, err := config.ConfigComment(cfg)
		if err != nil {
			return fmt.Errorf("marshal default config: %w", err)
		}

		err = ioutil.WriteFile(cfgPath, content, 0644)
		if err != nil {
			return fmt.Errorf("write config file: %w", err)
		}

		log.Info("init done")

		return nil
	},
}
