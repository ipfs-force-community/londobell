package aggregators

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

// FanoutRequestGate 是「重活扇出端点」的请求级并发闸（B 方案）。
//
// 这些端点（trace_for_message / blocks_for_message / hash_by_messagecid /
// child_transfers_for_message）每个请求都会向全部冷库扇出一轮完整聚合，
// 是内存峰值的主要来源。闸门满了直接 503 快速失败：不做一半工作、不排队等待。
//
// 默认关闭（limit=0）⇒ 行为与改前完全一致；开启后超限请求返回 503 + JSON 错误体，
// 是**显式拒绝**而不是静默截断。开关见 common.Config.FanoutRequestConcurrency /
// 环境变量 LONDOBELL_FANOUT_REQUEST_CONCURRENCY。
func FanoutRequestGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		release, err := multiquery.AcquireFanoutRequest()
		if err != nil {
			log.With("middleware", "FanoutRequestGate").Warnf("reject %v: %v", c.Request.URL.Path, err)

			c.AbortWithStatusJSON(http.StatusServiceUnavailable, model.CommonRes{
				Code: model.Fail,
				Msg:  err.Error(),
			})
			return
		}

		defer release()

		c.Next()
	}
}
