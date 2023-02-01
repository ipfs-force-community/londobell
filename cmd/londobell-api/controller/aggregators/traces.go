package aggregators

import (
	"bytes"
	"encoding/json"
	"net/http"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
	"golang.org/x/net/context"
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
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, tracesAggregator, req, "ExecTrace")
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
		methodInfo, err := util.LookupMethodInfo(trace.Epoch, abi.MethodNum(trace.Method), trace.From, trace.To, trace.Actor)
		if err != nil {
			alog.Warn(err)
		}

		if !trace.ParamsBson.IsZero() {
			params := methodInfo.ParamObj()
			if params != nil {
				err = params.UnmarshalCBOR(bytes.NewBuffer(trace.ParamsBson.Data))
				if err != nil {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}

				paramsByte, err := json.Marshal(params)
				if err != nil {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}
				trace.Params = string(paramsByte)
			}
		}

		if !trace.ReturnBson.IsZero() {
			returns := methodInfo.ReturnObj()
			if returns != nil {
				err = returns.UnmarshalCBOR(bytes.NewBuffer(trace.ReturnBson.Data))
				if err != nil {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}
				returnsByte, err := json.Marshal(returns)
				if err != nil {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}
				trace.Return = string(returnsByte)
			}
		}
	}

	res.Data = tracesRes
	c.JSON(http.StatusOK, res)
}
