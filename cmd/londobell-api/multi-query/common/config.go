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

	// 单请求「结果集/响应体」字节上限（见 multi-query/common/result_size.go）。
	// 分**两档**：查询类与元数据类。两档各自独立取值、独立来源，同一次
	// ApplyResultSizePolicy 一起落定。
	//
	//   MaxResultBytes（查询类：shard_query / query / response:* 等扇出与请求驱动的物化）
	//     0 = 内置默认(16 MiB)；>0 = 该字节数；<0 = 关闭上限(回退旧行为)。
	//
	//   MetadataMaxResultBytes（元数据类：metadata / segment_state，即进程启动与状态
	//   刷新必须全量加载的元数据）
	//     0 = 内置默认(256 MiB)；>0 = 该字节数；<0 = 关闭上限。
	//     为什么单独一档：元数据是**启动自身依赖**的数据，用查询档的 16 MiB 去卡它会让
	//     进程起不来（2026-09-30 实测：启动加载 segment_state = 16,777,354 字节，
	//     被 16 MiB 默认上限拒掉 ⇒ 聚合器 exit 1 崩溃循环）。元数据随链增长
	//     （actor/method 数量），所以默认值取实测值的 ~16×。
	//
	// 任何一次 `cur.All()` 物化累计超过**该 op 所属档**的上限即**显式报错**
	// （*ResultTooLargeError，util.ReturnOnErr 映射成 HTTP 5xx），绝不静默截断、
	// 绝不返回部分数据；累计达到上限的 80% 会打一条 warn 日志（可观测告警）。
	// 环境变量 LONDOBELL_MAX_RESULT_BYTES / LONDOBELL_METADATA_MAX_RESULT_BYTES
	// （字节，<=0 关闭）优先级更高。
	// 配置文件改动经 30s 配置巡检热生效（无需环境变量、无需重启）；环境变量需要重启进程。
	// 生效值与来源在启动/每次热重载时打一行 `result size policy applied: ...` 日志。
	MaxResultBytes         int64
	MetadataMaxResultBytes int64
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
