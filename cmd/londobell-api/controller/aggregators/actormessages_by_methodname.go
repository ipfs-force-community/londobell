package aggregators

import (
	"encoding/json"
	"net/http"

	"context"

	"github.com/gin-gonic/gin"
	pool_monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetActorMessagesByMethodName(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetActorMessagesByMethodName")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	req.Addr, err = GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetTotalCountForActorMsgByMethodName(ctx, req.Addr, req.MethodName, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.ActorMethodStates
	}

	var messagesByMethodName []model.MessageByMethodName
	// multi dbs query
	{
		multiResult, err := multiquery.MultiBiSearch(ctx, req.Index*req.Limit, req.Limit, countUtils, pool_monitor.GetActorMessagesByMethodNameNoskipAggregator(),
			pool_monitor.GetCountOfActorMessagesByMethodNameAggregator(), req, "ActorMessage", multiquery.ActorMethodStates)
		//multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.ActorMethodStates, countUtils, actorMessagesByMethodNameAggregator, req, "ActorMessage")
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

		err = json.Unmarshal(rawByte, &messagesByMethodName)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.MessagesByMethodNameRes{TotalCount: totalCount, MessagesByMethodName: messagesByMethodName}
	c.JSON(http.StatusOK, res)
}
