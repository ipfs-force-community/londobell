package aggregators

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetLatestTimeOfTrace(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetLatestTimeOfTrace")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	api := fullnode.API.GetAppropriateAPI()
	addrs, err := GetAllAddrs(ctx, req.Addr, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addrs = addrs

	pipe, err := util.Parse(model.Ctx{StartEpoch: 0, EndEpoch: math.MaxInt64, Addrs: req.Addrs, Sort: -1}, string(timeOfTraceAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var latestTimeOfTraceRes []model.TimeOfTraceRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiUnionQuery(ctx, pipe, countUtils, "ExecTrace")
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

		err = json.Unmarshal(rawByte, &latestTimeOfTraceRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	// 逆序排序
	sort.Slice(latestTimeOfTraceRes, func(i, j int) bool {
		return latestTimeOfTraceRes[i].Epoch > latestTimeOfTraceRes[j].Epoch
	})

	if len(latestTimeOfTraceRes) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = latestTimeOfTraceRes[0]
	c.JSON(http.StatusOK, res)
}
