package aggregators

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetTraces(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTraces")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var tracesRes []*model.TraceRes
	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, common2.TracesAggregator, req, "ExecTrace")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) == 0 {
			c.JSON(http.StatusOK, res)
			return
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &tracesRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	for _, trace := range tracesRes {
		methodInfo, _ := util.LookupMethodInfo(trace.Epoch, abi.MethodNum(trace.Method), trace.From, trace.To, trace.Actor)
		//if err != nil {
		//	alog.Warn(err)
		//}

		if !trace.ParamsBson.IsZero() {
			params := methodInfo.ParamObj()
			if params != nil {
				err = params.UnmarshalCBOR(bytes.NewBuffer(trace.ParamsBson.Data))
				if err != nil {
					trace.Params = "0x" + hex.EncodeToString(trace.ParamsBson.Data)
				} else {
					paramsByte, err := json.Marshal(params)
					if err != nil {
						// 链上参数里可能带「非最小编码」的 bitfield（RLE+ 尾部多 0x00）：
						// json.Marshal 会因此失败。退化为 hex 原始字节即可，
						// 不能让一条坏数据把整段高度的查询打成 500。
						alog.Warnf("marshal params fallback to hex (epoch=%d, method=%d, cid=%s): %v",
							trace.Epoch, trace.Method, trace.Cid, err)
						trace.Params = "0x" + hex.EncodeToString(trace.ParamsBson.Data)
					} else {
						trace.Params = string(paramsByte)
					}
				}
			}
		}

		if !trace.ReturnBson.IsZero() {
			returns := methodInfo.ReturnObj()
			if returns != nil {
				err = returns.UnmarshalCBOR(bytes.NewBuffer(trace.ReturnBson.Data))
				if err != nil {
					trace.Return = "0x" + hex.EncodeToString(trace.ReturnBson.Data)
				} else {
					returnsByte, err := json.Marshal(returns)
					if err != nil {
						// 同上：链上返回值里也可能带非最小编码的 bitfield。
						// 退化为 hex 原始字节，避免整段高度查询失败。
						alog.Warnf("marshal return fallback to hex (epoch=%d, method=%d, cid=%s): %v",
							trace.Epoch, trace.Method, trace.Cid, err)
						trace.Return = "0x" + hex.EncodeToString(trace.ReturnBson.Data)
					} else {
						trace.Return = string(returnsByte)
					}
				}
			}
		}
	}

	res.Data = tracesRes
	c.JSON(http.StatusOK, res)
}
