package multiquery

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/node/config"
)

type DBCollectionsConfigMgr struct {
	Cfg                   Config
	DBCollectionsMap      map[string]Collections
	DBCollectionsConfigLk sync.Mutex
}

func NewDBCollectionsConfigMgr(cfg Config) *DBCollectionsConfigMgr {
	return &DBCollectionsConfigMgr{
		Cfg:              cfg,
		DBCollectionsMap: make(map[string]Collections),
	}
}

// cold dbs每次新增就立即运行state存储；formal定时运行state存储；tmp每次都运行state存储
type Config struct {
	Colds          []DB
	Formal         DB
	Tmp            DB
	LastModifyTime int64
}

type DB struct {
	URL    string
	DBName string
}

func NewDB(url, name string) DB {
	return DB{
		URL:    url,
		DBName: name,
	}
}

func (db DB) Url() string {
	return db.URL
}

func (db DB) Name() string {
	return db.DBName
}

func (db DB) IsInvalidDB() bool {
	return db.Url() == "" || db.Name() == ""
}

func (db DB) Equals(o DB) bool {
	return db.Url() == o.Url() && db.Name() == o.Name()
}

func DefaultConfig() Config {
	colds := make([]DB, 0)
	return Config{
		Colds:          colds,
		Formal:         DB{},
		Tmp:            DB{},
		LastModifyTime: time.Now().Unix(),
	}
}

func WriteToConfig(cfgPath string, cfg Config) error {
	content, err := config.ConfigUpdate(cfg, nil, false)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}

	err = ioutil.WriteFile(cfgPath, content, 0644)
	if err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
