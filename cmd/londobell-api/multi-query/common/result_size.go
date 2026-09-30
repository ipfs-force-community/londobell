package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// ---------------------------------------------------------------------------
// 单请求「结果集 / 响应体」字节上限（两档：查询类 / 元数据类）
//
// 背景（2026-09-30 事故）：聚合器给分片查询加了全局并发闸（fanout_gate.go），
// 但闸门默认 limit=256 高于线上实际在飞连接数（~180），所以事故中闸门**从未触发**；
// 真正把 RSS 顶到 cgroup 上限的是「单请求的数据量 × 分配 churn」——
// 一次 `MultiPagingQuery(index=0, limit=0)` 会把整段（管道 limit=MaxInt64）全量物化，
// 单个请求就能吃掉几十上百 MB，180 个并发在飞请求叠加就把 RSS 从 914MB 推到 8.37GB。
//
// 因此需要一个**与并发数无关**的每请求上限：不管有多少请求在飞，任何一次
// `cur.All()` 物化的结果集都不允许超过 N 字节；超了立刻**显式报错**
// （*ResultTooLargeError），绝不静默截断、绝不返回部分数据。
//
// 为什么是**两档**上限（2026-09-30 首次上线后被生产判活拦下的故障）：
// 第一版把「查询类」与「元数据类」物化共用一个 16 MiB 上限，结果聚合器在启动时
// 就被自己的元数据加载拒掉、exit 1 崩溃循环：
//
//	result set too large: op=segment_state, limit=16777216 bytes,
//	exceeded at 16777354 bytes; refusing to materialize a partial result
//
// 也就是说启动**自身依赖**的元数据（segment_state，实测 16,777,354 字节）只比 16 MiB
// 默认上限多 138 字节，就被上限拒了。这两类物化的性质完全不同：
//
//   - 查询类（shard_query / query / response:*）：由**外部请求**触发、与流量成正比、
//     单次结果大小不可预测，是 OOM 的直接来源 ⇒ 必须用紧的上限（16 MiB）封顶；
//   - 元数据类（metadata / segment_state）：进程**启动与状态刷新**必须全量加载的
//     元数据（各库高度区间、各 actor/method 的 segment 覆盖），是**正确性的前提**，
//     不是可裁剪的查询结果；它只随链增长（actor/method 数量），量级可预测
//     ⇒ 给一档明显更高的上限（256 MiB），既保留封顶（防病态），又不把启动打死。
//
// 这个文件提供：
//   - DefaultMaxResultBytes（查询类，16 MiB）与 DefaultMetadataMaxResultBytes
//     （元数据类，256 MiB），推导见各自常量注释；
//   - ResultTooLargeError：可识别错误类型（util.ReturnOnErr 会把它映射成 HTTP 5xx）；
//   - BoundedAll：`cur.All(ctx, dst)` 的等价替代 —— 语义逐字节一致（含顺序），
//     只是改成「带上限的迭代」：累计物化的 BSON 字节超限即报错；
//     上限按 op 自动路由到查询档或元数据档（见 LimitForOp）；
//   - 策略解析：环境变量 > 配置文件 > 内置默认；<=0 = 关闭上限（回退旧行为）；
//   - 接近上限的可观测告警（ResultSizeWarnHook，默认打 warn 日志）。
// ---------------------------------------------------------------------------

const (
	// DefaultMaxResultBytes 是**查询类**物化的默认字节上限（扇出 / 请求驱动的查询）。
	//
	// 推导（与已有的并发闸配套，不是拍脑袋）：
	//   - 进程 GOMEMLIMIT = 4 GiB；出站分片查询闸门默认 limit = 256
	//     （见 multi-query/fanout_gate.go 的 DefaultShardQueryConcurrency）；
	//   - 把内存预算按闸门槽位均分：4 GiB / 256 = 16 MiB/槽位。
	//     即「闸门满负荷时，每个在飞物化最多用掉自己那份公平配额，合计不超过预算」。
	//   - 事故期实测单个在飞查询的 in-use 仅约 1 MB（192 MB / ~180 在飞），
	//     所以 16 MiB 对正常流量是 ~16× 余量 —— 只有病态的「全量物化」才会触发，
	//     而那正是事故驱动因素。
	// 实际 Go/BSON 开销约为原始字节的 2×，所以「全部 256 槽位同时打满」是理论最坏
	// 上界（~2× GOMEMLIMIT）；上限是每请求安全网，闸门才是总量约束器。
	// 运维可按需下调（配置文件 MaxResultBytes / 环境变量 LONDOBELL_MAX_RESULT_BYTES）。
	//
	// 注意：**不要**用这一档去卡元数据类物化 —— 启动会因此失败（见文件头注释）。
	DefaultMaxResultBytes int64 = 16 << 20

	// DefaultMetadataMaxResultBytes 是**元数据类**物化（op = metadata / segment_state）
	// 的默认字节上限：进程启动与状态刷新必须全量加载的元数据。
	//
	// 取值依据（实测，不是拍脑袋）：
	//   - 生产实测：聚合器启动时必须加载的 segment_state 元数据 = 16,777,354 字节
	//     （~16.0 MiB），即第一版 16 MiB 上限被它超了 138 字节 ⇒ 进程 exit 1 崩溃循环；
	//   - 该量级随链增长（actor 数 × method 数 × segment 数），不是常量，
	//     所以上限必须是现实值的**数倍**而不是贴身；256 MiB ≈ 实测值的 16×，
	//     给「元数据再涨一个数量级」留出空间；
	//   - 元数据加载是**启动一次性 / 缓存失效时**发生（不是每请求），即使达到 256 MiB
	//     也是短时、可回收的；它不会像查询类物化那样随并发数线性叠加
	//     （RefreshState 整轮还被出站闸门 withShardSlot 限流，见 dbstatemanager.go）；
	//   - 同时保留「封顶」的意义：元数据若真涨到 256 MiB 以上，说明分片元数据模型
	//     需要重构（应该按 actor 分页/懒加载），那时**显式报错**比悄悄 OOM 更好。
	// 运维可按需调整（配置文件 MetadataMaxResultBytes /
	// 环境变量 LONDOBELL_METADATA_MAX_RESULT_BYTES）；<=0 = 关闭该档上限。
	DefaultMetadataMaxResultBytes int64 = 256 << 20

	// ResultSizeWarnRatio 是「接近上限」告警的阈值比例：单次物化累计字节达到
	// 生效上限的该比例时，通过 ResultSizeWarnHook 发出可观测告警（默认 warn 日志）。
	// 取 0.8：留 20% 余量给「还没告警就超了」的最后一跳，且正常流量（~1 MB / 16 MiB
	// = 6%）绝不会噪声性触发；元数据档（实测 16.7 MB / 256 MiB = 6.5%）同样安静。
	ResultSizeWarnRatio = 0.8

	// MaxResultBytesEnv 环境变量（字节）：查询类上限；>0 = 该上限；<=0 = 关闭（回退旧行为）。
	MaxResultBytesEnv = "LONDOBELL_MAX_RESULT_BYTES"

	// MetadataMaxResultBytesEnv 环境变量（字节）：元数据类上限；>0 = 该上限；<=0 = 关闭。
	MetadataMaxResultBytesEnv = "LONDOBELL_METADATA_MAX_RESULT_BYTES"
)

// 元数据类物化的 op 名（与各调用点 BoundedAll 的 op 参数一致）。
// 这些是「进程启动 / 状态刷新必须全量加载」的元数据，走 DefaultMetadataMaxResultBytes 档；
// 其余 op（shard_query / query / boundary / response:* ...）走 DefaultMaxResultBytes 档。
const (
	OpMetadata     = "metadata"      // totalcount.go：各 actor/method 的元数据聚合
	OpSegmentState = "segment_state" // segment/persist.go：DBState/BlockState/ActorState... 全量加载
)

var log = logging.Logger("multi-query/result-size")

// IsMetadataOp 判断一个物化 op 是否属于「元数据类」（走独立的高上限档）。
func IsMetadataOp(op string) bool {
	switch op {
	case OpMetadata, OpSegmentState:
		return true
	default:
		return strings.HasPrefix(op, OpMetadata+":")
	}
}

// ResultTooLargeError 表示「一次结果集物化超过了配置的字节上限」。
//
// 它是一个**显式错误**：调用方必须把它当失败上报，绝不能返回已经物化的那部分数据
// （那会静默截断结果）。util.ReturnOnErr 会把它映射成 HTTP 5xx。
type ResultTooLargeError struct {
	// Op 标识是哪一类物化触发的（shard_query / metadata / segment_state ...）。
	Op string
	// Limit 是生效的字节上限。
	Limit int64
	// Actual 是触发时报出的累计字节数（>= Limit）。
	Actual int64
}

func (e *ResultTooLargeError) Error() string {
	return fmt.Sprintf("result set too large: op=%s, limit=%d bytes, exceeded at %d bytes; "+
		"refusing to materialize a partial result", e.Op, e.Limit, e.Actual)
}

// IsResultTooLarge 判断 err 是否是（或包装了）ResultTooLargeError。
func IsResultTooLarge(err error) bool {
	var target *ResultTooLargeError
	return errors.As(err, &target)
}

// maxResultBytes 是当前生效的**查询类**字节上限；<=0 表示关闭（回退旧行为）。
// 进程启动与配置热重载时通过 ApplyResultSizePolicy 落定。
var maxResultBytes atomic.Int64

// metadataMaxResultBytes 是当前生效的**元数据类**字节上限；<=0 表示关闭。
// 与 maxResultBytes 同源（同一次 ApplyResultSizePolicy 落定），但独立取值。
var metadataMaxResultBytes atomic.Int64

func init() {
	maxResultBytes.Store(DefaultMaxResultBytes)
	metadataMaxResultBytes.Store(DefaultMetadataMaxResultBytes)
}

// SetMaxResultBytes 直接设置**查询类**上限（测试 / 运维热调钩子）。n<=0 = 关闭上限。
func SetMaxResultBytes(n int64) {
	if n < 0 {
		n = 0
	}
	maxResultBytes.Store(n)
}

// MaxResultBytes 返回当前生效的**查询类**上限（<=0 = 关闭）。
func MaxResultBytes() int64 {
	return maxResultBytes.Load()
}

// SetMetadataMaxResultBytes 直接设置**元数据类**上限（测试 / 运维热调钩子）。n<=0 = 关闭。
func SetMetadataMaxResultBytes(n int64) {
	if n < 0 {
		n = 0
	}
	metadataMaxResultBytes.Store(n)
}

// MetadataMaxResultBytes 返回当前生效的**元数据类**上限（<=0 = 关闭）。
func MetadataMaxResultBytes() int64 {
	return metadataMaxResultBytes.Load()
}

// LimitForOp 返回某个物化 op 生效的字节上限（<=0 = 关闭）。
// 元数据类 op 走独立的元数据档，其余走查询档 —— 这是第一版启动崩溃的修复点：
// 启动必须加载的元数据不再被查询档的 16 MiB 上限拒掉。
func LimitForOp(op string) int64 {
	if IsMetadataOp(op) {
		return metadataMaxResultBytes.Load()
	}
	return maxResultBytes.Load()
}

// NearLimit 判断单次物化用量是否已到「接近上限」的告警阈值（limit<=0 = 关闭 ⇒ 永不告警）。
func NearLimit(used, limit int64) bool {
	if limit <= 0 {
		return false
	}
	return float64(used) >= ResultSizeWarnRatio*float64(limit)
}

// ResultSizeWarnHook 在单次物化累计字节达到生效上限的 ResultSizeWarnRatio 时被调用一次
// （默认实现打 warn 日志）。它存在的意义：上限是「拒绝服务」的硬闸，运维需要在**被拒之前**
// 看到「元数据/结果正在逼近上限」的信号（元数据随链增长 ⇒ 需要提前规划，而不是等崩溃循环）。
// 测试可替换它来断言告警确实发出（务必用 defer 还原）。
var ResultSizeWarnHook = func(op string, used, limit int64) {
	log.Warnf("result set approaching byte limit: op=%s used=%d bytes limit=%d bytes (%.1f%% of limit), "+
		"kind=%s; raise the limit (config MetadataMaxResultBytes/MaxResultBytes or env %s/%s) before it starts being rejected",
		op, used, limit, 100*float64(used)/float64(limit), resultSizeKind(op), MetadataMaxResultBytesEnv, MaxResultBytesEnv)
}

func resultSizeKind(op string) string {
	if IsMetadataOp(op) {
		return "metadata"
	}
	return "query"
}

// ---------------------------------------------------------------------------
// 策略解析：环境变量 > 配置文件 > 内置默认
// ---------------------------------------------------------------------------

// ResultSizePolicy 是一次策略解析的完整结果（两档上限 + 各自来源），用于日志可验证性。
// Source 取值：default / config / config(disabled) / env / env(disabled)。
type ResultSizePolicy struct {
	QueryLimit     int64
	QuerySource    string
	MetadataLimit  int64
	MetadataSource string
}

// String 输出一行可直接 grep 的生效值（带来源）。
func (p ResultSizePolicy) String() string {
	return fmt.Sprintf("query_limit=%d (%s, %.2f MiB), metadata_limit=%d (%s, %.2f MiB)",
		p.QueryLimit, p.QuerySource, float64(p.QueryLimit)/(1<<20),
		p.MetadataLimit, p.MetadataSource, float64(p.MetadataLimit)/(1<<20))
}

// resolveLimit 解析单档上限：环境变量 > 配置文件 > 内置默认。返回 (limit, source)。
//   - env 合法且 >0 ⇒ (n, "env")；env 合法且 <=0 ⇒ (0, "env(disabled)")；env 非法 ⇒ 忽略；
//   - cfg >0 ⇒ (cfg, "config")；cfg <0 ⇒ (0, "config(disabled)")；cfg =0 ⇒ (def, "default")。
func resolveLimit(cfgVal int64, envName string, def int64) (int64, string) {
	limit, source := def, "default"
	switch {
	case cfgVal > 0:
		limit, source = cfgVal, "config"
	case cfgVal < 0:
		limit, source = 0, "config(disabled)"
	}

	if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			if n <= 0 {
				limit, source = 0, "env(disabled)"
			} else {
				limit, source = n, "env"
			}
		}
	}

	return limit, source
}

// ResolveResultSizePolicyFull 计算两档生效上限（含来源）。每次调用现算，
// 所以改配置文件（30s 巡检热生效）与改环境变量（需重启）都能生效。
func ResolveResultSizePolicyFull(cfg Config) ResultSizePolicy {
	qLimit, qSource := resolveLimit(cfg.MaxResultBytes, MaxResultBytesEnv, DefaultMaxResultBytes)
	mLimit, mSource := resolveLimit(cfg.MetadataMaxResultBytes, MetadataMaxResultBytesEnv, DefaultMetadataMaxResultBytes)

	return ResultSizePolicy{
		QueryLimit:     qLimit,
		QuerySource:    qSource,
		MetadataLimit:  mLimit,
		MetadataSource: mSource,
	}
}

// ResolveResultSizePolicy 计算生效的**查询类**上限：环境变量 > 配置文件 > 内置默认。
//   - cfg.MaxResultBytes > 0 ⇒ 用该值；
//   - cfg.MaxResultBytes < 0 ⇒ 关闭（0）；
//   - cfg.MaxResultBytes = 0 ⇒ 内置默认（16 MiB）。
//
// 环境变量 LONDOBELL_MAX_RESULT_BYTES：>0 = 该值；<=0 = 关闭。
func ResolveResultSizePolicy(cfg Config) int64 {
	return ResolveResultSizePolicyFull(cfg).QueryLimit
}

// ResolveMetadataResultSizePolicy 计算生效的**元数据类**上限：环境变量 > 配置文件 > 内置默认。
//   - cfg.MetadataMaxResultBytes > 0 ⇒ 用该值；
//   - cfg.MetadataMaxResultBytes < 0 ⇒ 关闭（0）；
//   - cfg.MetadataMaxResultBytes = 0 ⇒ 内置默认（256 MiB）。
//
// 环境变量 LONDOBELL_METADATA_MAX_RESULT_BYTES：>0 = 该值；<=0 = 关闭。
func ResolveMetadataResultSizePolicy(cfg Config) int64 {
	return ResolveResultSizePolicyFull(cfg).MetadataLimit
}

// ApplyResultSizePolicy 从配置计算并应用两档上限，并把生效值与来源打成一行日志。
// 进程启动（FirstLoad）与配置热重载（SetConfig）时调用 —— 运维据此可以在日志里
// 直接确认「配置文件旋钮到底有没有生效」，不必依赖环境变量或猜。
func ApplyResultSizePolicy(cfg Config) {
	p := ResolveResultSizePolicyFull(cfg)

	SetMaxResultBytes(p.QueryLimit)
	SetMetadataMaxResultBytes(p.MetadataLimit)

	log.Infof("result size policy applied: %s; ops: metadata=%v -> metadata_limit, others -> query_limit",
		p.String(), []string{OpMetadata, OpSegmentState})

	if p.QueryLimit > 0 && p.MetadataLimit > 0 && p.MetadataLimit < p.QueryLimit {
		// 元数据档低于查询档几乎肯定是配置写反了（元数据是启动依赖，必须更宽松）。
		log.Warnf("metadata result limit (%d) is BELOW query limit (%d); startup metadata load may be rejected. "+
			"metadata ops are %v", p.MetadataLimit, p.QueryLimit, []string{OpMetadata, OpSegmentState})
	}
}

// ---------------------------------------------------------------------------
// BoundedAll：带上限的 cur.All 等价物
// ---------------------------------------------------------------------------

// BoundedAll 是 `cur.All(ctx, dst)` 的等价替代：把整个结果集读进 dst，
// 但累计物化的 BSON 字节超过该 op 生效的上限（见 LimitForOp）时返回 *ResultTooLargeError。
//
// 语义保证（与 cur.All 逐字节一致，含顺序）：
//   - 逐文档按游标顺序解码、按同样顺序 append；解码器与 cur.All 相同（bson 默认解码）；
//   - 上限关闭（<=0）时不设任何限制，行为与 cur.All 完全一致（回退旧行为）；
//   - 超限时 dst 保持不变（由调用方初始化为零值），**绝不返回部分结果**；
//   - 累计字节达到生效上限的 ResultSizeWarnRatio 时，通过 ResultSizeWarnHook 发出
//     一次可观测告警（默认 warn 日志），但在真正超限之前**不**报错。
//
// 字节口径：累计 `len(cur.Current)`，即 mongo 返回的原始 BSON 文档字节数。
// 这是稳定、与驱动无关的度量；Go 侧 map/slice 开销约为其 2×（默认上限已计入）。
func BoundedAll[T any](ctx context.Context, cur *mongo.Cursor, dst *[]T, op string) error {
	if cur == nil {
		return errors.New("BoundedAll: nil cursor")
	}

	limit := LimitForOp(op)

	res := make([]T, 0)
	var total int64
	warned := false

	for cur.Next(ctx) {
		total += int64(len(cur.Current))
		if limit > 0 && total > limit {
			// 先把游标关掉再报错，避免连接/内存被游标拖住。
			_ = cur.Close(ctx)
			return &ResultTooLargeError{Op: op, Limit: limit, Actual: total}
		}

		if !warned && NearLimit(total, limit) {
			warned = true
			ResultSizeWarnHook(op, total, limit)
		}

		var v T
		if err := cur.Decode(&v); err != nil {
			_ = cur.Close(ctx)
			return err
		}
		res = append(res, v)
	}

	if err := cur.Err(); err != nil {
		_ = cur.Close(ctx)
		return err
	}

	*dst = res
	return nil
}
