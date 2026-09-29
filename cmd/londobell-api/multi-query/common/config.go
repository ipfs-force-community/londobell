package common

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
	Colds                   []DB
	Formal                  DB
	Tmp                     DB
	LastModifyTime          int64
	BatchSegmentInsertLimit int

	// 元数据扇出缓存开关与 TTL（refresh() 产出的「各库高度区间 / 各库总条数」）。
	// 零值 = 开启缓存 + 默认 30s：配置文件里不写这两个字段就是默认行为。
	//   DisableMetaCache=true        ⇒ 关闭缓存，回退到「每个请求都重新向全部库扇出一遍」；
	//   MetaCacheTTLSeconds > 0      ⇒ 用该秒数做 TTL；<=0 且未 Disable ⇒ 默认 30s。
	// 环境变量 LONDOBELL_METADATA_CACHE_TTL（秒，<=0 关闭）与
	// LONDOBELL_METADATA_CACHE_DISABLED=1 优先级更高。配置文件改动经 30s 的配置巡检热生效，
	// 环境变量需要重启进程。
	DisableMetaCache    bool
	MetaCacheTTLSeconds int64

	// 出站分片查询的全局并发上限与有界等待（见 multi-query/fanout_gate.go）。
	//   ShardQueryConcurrency: 0 = 内置默认(256)；>0 = 该值；<0 = 关闭闸门(回退旧行为)。
	//   ShardQueryWaitSeconds: 0 = 内置默认(5s)；>0 = 该秒数；<0 = 无限等待(直到请求 ctx 取消)。
	// 拿不到令牌且等待超时会**显式报错**（绝不返回空/部分结果）。
	// 环境变量 LONDOBELL_SHARD_QUERY_CONCURRENCY / LONDOBELL_SHARD_QUERY_WAIT_SECONDS 优先级更高。
	// 配置文件改动经 30s 配置巡检热生效；环境变量需要重启进程。
	ShardQueryConcurrency int
	ShardQueryWaitSeconds int

	// 扇出家族端点(trace_for_message / blocks_for_message / hash_by_messagecid /
	// child_transfers_for_message)的请求级并发上限：0 = 关闭(默认)，>0 = 超限即 503 快速失败。
	// 环境变量 LONDOBELL_FANOUT_REQUEST_CONCURRENCY 优先级更高。
	FanoutRequestConcurrency int
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

func EmptyDB() DB {
	return DB{
		URL:    "",
		DBName: "",
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
		Colds:                   colds,
		Formal:                  DB{},
		Tmp:                     DB{},
		LastModifyTime:          time.Now().Unix(),
		BatchSegmentInsertLimit: 16,
	}
}

func WriteToConfig(cfgPath string, cfg Config) error {
	content, err := config.ConfigUpdate(cfg, nil, config.Commented(false))
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}

	err = ioutil.WriteFile(cfgPath, content, 0644)
	if err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
