package aggregators

import (
	"encoding/json"
	"net/http"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetTestContractTransfer(c *gin.Context) {
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
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, monitor.GetTestContractTransferAggregator(), req, "ExecTrace")
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

	res.Data = tracesRes
	c.JSON(http.StatusOK, res)
}
