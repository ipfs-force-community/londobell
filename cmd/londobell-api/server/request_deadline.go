package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

// 默认的请求硬上限。取值依据（2026-10-24 生产实测）：
// 正常请求 p99 < 1s，最重的账号类接口（all_methods_for_actor / messages_for_actor
// 对热点大地址）实测 25~41s，所以 60s 只砍「已经明显异常」的请求，不会误伤正常业务。
// 等 C3（MultiBiSearch 并行化）把那些接口降到秒级后，可以再往下调。
const defaultRequestTimeout = 60 * time.Second

// deadlineHits 统计 RequestDeadline 实际触发（写 504 或 handler 因超时返回）的次数，
// 由 /debug/runtime 暴露，用来验证请求级上限在生产上真的在生效。
var deadlineHits int64

func requestDeadlineHits() int64 { return atomic.LoadInt64(&deadlineHits) }

// RequestDeadline 给每个请求挂上硬 deadline，并让 handler 继承请求的 context。
//
// 为什么需要这件事：排查聚合器 OOM 时发现 105 个 handler 都是
// `context.WithCancel(context.Background())` 起的 ctx——服务端完全不知道客户端已经断开。
// 浏览器/curl 早就放弃了，服务端还在把整个分片扫完（我自己的探测就制造过两条
// 「客户端超时、服务端继续跑十几分钟」的请求）。父 ctx 换成 c.Request.Context() 之后，
// 客户端断开即取消；这里的 deadline 是兜底上限，防止任何单请求无限期占用内存与分片闸门。
//
// 关掉的方式：LONDOBELL_REQUEST_TIMEOUT=0（或负数），本进程不做 deadline，只继承客户端取消。
func RequestDeadline() gin.HandlerFunc {
	timeout := requestTimeoutFromEnv()
	log.Infof("request deadline policy applied: timeout=%s", timeoutText(timeout))

	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		// 只有「超时且响应一个字都还没写」时才补 504：
		// handler 已经写过响应体的（包括流式/分块）不再插手，避免破坏已发出的 JSON。
		if ctx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&deadlineHits, 1)
		}

		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, model.CommonRes{
				Code: model.Fail,
				Msg:  fmt.Sprintf("请求超过 %s 未完成，服务端已中断（LONDOBELL_REQUEST_TIMEOUT 可调整）", timeout),
			})
		}
	}
}

// requestTimeoutFromEnv 读 LONDOBELL_REQUEST_TIMEOUT，非法值退回默认（并告警，不静默）。
func requestTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("LONDOBELL_REQUEST_TIMEOUT"))
	if raw == "" {
		return defaultRequestTimeout
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Warnf("解析 LONDOBELL_REQUEST_TIMEOUT=%q 失败(%v)，使用默认值 %s", raw, err, defaultRequestTimeout)
		return defaultRequestTimeout
	}

	return d
}

func timeoutText(d time.Duration) string {
	if d <= 0 {
		return "off"
	}
	return d.String()
}
