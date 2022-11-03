package aggregators

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTraces(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTraces")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var tracesRes []*model.TraceRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch}, string(tracesAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &tracesRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	for _, trace := range tracesRes {
		methodInfo, err := util.LookupMethodInfo(trace.Epoch, abi.MethodNum(trace.Method), trace.From, trace.To, trace.Actor)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		if trace.Params != nil {
			params := methodInfo.ParamObj()
			if params != nil {
				err = params.UnmarshalCBOR(bytes.NewBuffer(trace.Params.(primitive.Binary).Data))
				if err != nil {
					util.ReturnOnErr(c, alog, err)
					return
				}

				paramsByte, err := json.Marshal(params)
				if err != nil {
					util.ReturnOnErr(c, alog, err)
					return
				}
				trace.Params = string(paramsByte)
			}
		}

		if trace.Return != nil {
			returns := methodInfo.ReturnObj()
			if returns != nil {
				err = returns.UnmarshalCBOR(bytes.NewBuffer(trace.Return.(primitive.Binary).Data))
				if err != nil {
					util.ReturnOnErr(c, alog, err)
					return
				}
				returnsByte, err := json.Marshal(returns)
				if err != nil {
					util.ReturnOnErr(c, alog, err)
					return
				}
				trace.Return = string(returnsByte)
			}
		}
	}

	res.Data = tracesRes
	c.JSON(http.StatusOK, res)
}
