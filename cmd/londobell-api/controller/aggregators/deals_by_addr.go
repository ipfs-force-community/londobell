package aggregators

import (
	"encoding/json"
	"math"
	"net/http"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetDealsByAddr(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetDealsByAddr")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	actorID, err := GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addr = actorID
	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetTotalCountForActorDeals(ctx, actorID, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.DealActorStates
	}

	if req.Index == 0 && req.Limit == 0 {
		req.Limit = math.MaxInt64
	}

	var deals []model.Deal

	// multi dbs query
	{
		multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.DealActorStates, countUtils, dealsByAddrAggregator, req, "NewDealProposal")
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

		err = json.Unmarshal(rawByte, &deals)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.DealsRes{TotalCount: totalCount, Deals: deals}
	c.JSON(http.StatusOK, res)
}
