package util

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
)

var ErrNotFound = fmt.Errorf("get wrong result, length of result shoule be one")

// ReturnOnErr 把 controller 的失败统一写回响应。
//
// 「结果集超上限」是**服务端拒绝**（不是业务失败），必须用 HTTP 5xx 表达，
// 而不是既有的 HTTP 200 + code=Fail —— 否则调用方会把「这个请求太大」当成
// 「没有数据」。这里对 *common.ResultTooLargeError 返回 500 + 可识别的错误消息。
func ReturnOnErr(c *gin.Context, err error) {
	if err != nil {
		res := model.CommonRes{}

		if common.IsResultTooLarge(err) {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusInternalServerError, res)
			return
		}

		if err == ErrNotFound {
			res.Code = model.NotFound
		} else {
			res.Code = model.Fail
		}

		res.Msg = err.Error()
		c.JSON(http.StatusOK, res) // todo: status code
		return
	}
}

// MarshalBounded 是 json.Marshal 的带上限替代，用于「响应体字节上限」。
//
// 语义与 json.Marshal 完全一致（同一编码器、同一字段顺序），只是序列化完成后
// 检查长度：超过上限返回 *common.ResultTooLargeError（显式错误，绝不截断），
// 上限<=0 时不做检查（回退旧行为）。
//
// 它给那些「结果集来自非分片数据源、因此没被 common.BoundedAll 覆盖」的
// 响应组装路径兜底（例如 GetTipSet / GetBlockHeaderByCid 的 json.Marshal）。
func MarshalBounded(v interface{}, op string) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	limit := common.MaxResultBytes()
	if limit > 0 && int64(len(raw)) > limit {
		return nil, &common.ResultTooLargeError{Op: op, Limit: limit, Actual: int64(len(raw))}
	}

	return raw, nil
}
