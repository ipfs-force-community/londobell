package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/node/config"

	"github.com/dtynn/londobell/racailum"
)

var cfgCmd = &cli.Command{
	Name: "cfg",
	Subcommands: []*cli.Command{
		cfgInitCmd,
	},
}

var cfgInitCmd = &cli.Command{
	Name:  "init",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		cfgPath, err := configPath(cctx)
		if err != nil {
			return err
		}

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

func configPath(cctx *cli.Context) (string, error) {
	dir, err := getRepoHomeDir(cctx)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config"), nil
}

func loadConfig(cctx *cli.Context) (racailum.Config, error) {
	cfgPath, err := configPath(cctx)
	if err != nil {
		return racailum.Config{}, err
	}

	cfg := racailum.DefaultConfig()
	_, err = config.FromFile(cfgPath, &cfg)
	if err != nil {
		return racailum.Config{}, err
	}

	return cfg, nil
}
