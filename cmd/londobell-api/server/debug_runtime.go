package server

import (
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

var processStart = time.Now()

// DebugRuntime 暴露进程级运行时读数，给看门狗与排障脚本一个稳定入口：
// 协程数、堆/栈、闸门计数、请求 deadline 命中数。
//
// 为什么需要它：2026-10-24 的事故里，内存从 1 GB 涨到 12.55 GB 只用了 90 秒，
// 但进程内部读数只能靠 kill -USR1 抓 profile 再离线解析，一次排障要十几分钟。
// 这里把「一眼能看出是不是协程栈爆炸」的几个数直接吐出来：
//   - goroutines / stack_inuse_mb：协程栈爆炸时这两个数一起飙（事故时 89 万协程 ≈ 6 GB 栈）；
//   - shard_gate.inflight/saturated：出站闸门是否被打满；
//   - deadline_hits：RequestDeadline 触发次数，用来验证请求级上限是否真的在生效。
//
// 安全：只放行回环与私网直连（判定用 RemoteAddr，**不看 X-Forwarded-For**——
// gin 默认信任代理头，公网客户端可以伪造 XFF 冒充内网）。
func DebugRuntime(c *gin.Context) {
	if !isLoopbackOrPrivate(c.Request.RemoteAddr) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "debug endpoint is limited to loopback/private sources",
		})
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.JSON(http.StatusOK, gin.H{
		"goroutines":     runtime.NumGoroutine(),
		"uptime_s":       int64(time.Since(processStart).Seconds()),
		"heap_inuse_mb":  m.HeapInuse / 1024 / 1024,
		"heap_alloc_mb":  m.HeapAlloc / 1024 / 1024,
		"heap_sys_mb":    m.HeapSys / 1024 / 1024,
		"stack_inuse_mb": m.StackInuse / 1024 / 1024,
		"num_gc":         m.NumGC,
		"shard_gate":     multiquery.ShardGateSnapshot(),
		"fanout_gate":    multiquery.FanoutRequestGateSnapshot(),
		"deadline_hits":  requestDeadlineHits(),
	})
}

// isLoopbackOrPrivate 判断 RemoteAddr（host:port）是否来自回环或私网。
func isLoopbackOrPrivate(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback() || ip.IsPrivate()
}
