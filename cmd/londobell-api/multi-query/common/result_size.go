package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/mongo"
)

// ---------------------------------------------------------------------------
// 单请求「结果集 / 响应体」字节上限
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
// 这个文件提供：
//   - DefaultMaxResultBytes：默认上限（推导见常量注释）；
//   - ResultTooLargeError：可识别错误类型（util.ReturnOnErr 会把它映射成 HTTP 5xx）；
//   - BoundedAll：`cur.All(ctx, dst)` 的等价替代 —— 语义逐字节一致（含顺序），
//     只是改成「带上限的迭代」：累计物化的 BSON 字节超限即报错；
//   - 策略解析：环境变量 > 配置文件 > 内置默认；<=0 = 关闭上限（回退旧行为）。
// ---------------------------------------------------------------------------

const (
	// DefaultMaxResultBytes 是单次结果集物化的默认字节上限。
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
	// 运维可按需下调（配置文件 MaxResultBytes / 环境变量）。
	DefaultMaxResultBytes int64 = 16 << 20

	// MaxResultBytesEnv 环境变量（字节）：>0 = 该上限；<=0 = 关闭上限（回退旧行为）。
	MaxResultBytesEnv = "LONDOBELL_MAX_RESULT_BYTES"
)

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

// maxResultBytes 是当前生效的字节上限；<=0 表示关闭（回退旧行为）。
// 进程启动与配置热重载时通过 ApplyResultSizePolicy 落定。
var maxResultBytes atomic.Int64

func init() {
	maxResultBytes.Store(DefaultMaxResultBytes)
}

// SetMaxResultBytes 直接设置上限（测试 / 运维热调钩子）。n<=0 = 关闭上限。
func SetMaxResultBytes(n int64) {
	if n < 0 {
		n = 0
	}
	maxResultBytes.Store(n)
}

// MaxResultBytes 返回当前生效的上限（<=0 = 关闭）。
func MaxResultBytes() int64 {
	return maxResultBytes.Load()
}

// ResolveResultSizePolicy 计算生效上限：环境变量 > 配置文件 > 内置默认。
// 每次调用现算，所以改配置文件（30s 巡检热生效）与改环境变量（需重启）都能生效。
//   - cfg.MaxResultBytes > 0 ⇒ 用该值；
//   - cfg.MaxResultBytes < 0 ⇒ 关闭（0）；
//   - cfg.MaxResultBytes = 0 ⇒ 内置默认（16 MiB）。
//
// 环境变量 LONDOBELL_MAX_RESULT_BYTES：>0 = 该值；<=0 = 关闭。
func ResolveResultSizePolicy(cfg Config) int64 {
	limit := DefaultMaxResultBytes
	switch {
	case cfg.MaxResultBytes > 0:
		limit = cfg.MaxResultBytes
	case cfg.MaxResultBytes < 0:
		limit = 0
	}

	if v := strings.TrimSpace(os.Getenv(MaxResultBytesEnv)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			if n <= 0 {
				limit = 0
			} else {
				limit = n
			}
		}
	}

	return limit
}

// ApplyResultSizePolicy 从配置计算并应用上限。进程启动与配置重载时调用。
func ApplyResultSizePolicy(cfg Config) {
	SetMaxResultBytes(ResolveResultSizePolicy(cfg))
}

// BoundedAll 是 `cur.All(ctx, dst)` 的等价替代：把整个结果集读进 dst，
// 但累计物化的 BSON 字节超过生效上限时返回 *ResultTooLargeError。
//
// 语义保证（与 cur.All 逐字节一致，含顺序）：
//   - 逐文档按游标顺序解码、按同样顺序 append；解码器与 cur.All 相同（bson 默认解码）；
//   - 上限关闭（<=0）时不设任何限制，行为与 cur.All 完全一致（回退旧行为）；
//   - 超限时 dst 保持不变（由调用方初始化为零值），**绝不返回部分结果**。
//
// 字节口径：累计 `len(cur.Current)`，即 mongo 返回的原始 BSON 文档字节数。
// 这是稳定、与驱动无关的度量；Go 侧 map/slice 开销约为其 2×（默认上限已计入）。
func BoundedAll[T any](ctx context.Context, cur *mongo.Cursor, dst *[]T, op string) error {
	if cur == nil {
		return errors.New("BoundedAll: nil cursor")
	}

	limit := maxResultBytes.Load()

	res := make([]T, 0)
	var total int64

	for cur.Next(ctx) {
		total += int64(len(cur.Current))
		if limit > 0 && total > limit {
			// 先把游标关掉再报错，避免连接/内存被游标拖住。
			_ = cur.Close(ctx)
			return &ResultTooLargeError{Op: op, Limit: limit, Actual: total}
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
