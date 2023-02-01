package multiquery

import (
	"fmt"
	"path/filepath"

	"github.com/filecoin-project/lotus/node/config"
)

func LoadRaConfig(rpath RepoPath) (Config, error) {
	cfgPath := ConfigFilePath(rpath)
	cfg := DefaultConfig()
	_, err := config.FromFile(cfgPath, &cfg) // todo: config不适合当前数据库配置需求
	if err != nil {
		return Config{}, fmt.Errorf("read config from file %s: %w", cfgPath, err)
	}

	return cfg, nil
}

func ConfigFilePath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "config")
}
