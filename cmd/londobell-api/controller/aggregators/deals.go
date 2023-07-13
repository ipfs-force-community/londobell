package aggregators

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetDeals(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetDeals")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetDealRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.DealState
	}

	if req.Index == 0 && req.Limit == 0 {
		req.Limit = math.MaxInt64
	}

	var deals []model.Deal

	// multi dbs query
	{
		multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.DealState, countUtils, dealsAggregator, req, "DealProposal")
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

func GetTotalCountForAllDeals(ctx context.Context, cols common2.Collections, epoch abi.ChainEpoch) (int64, error) {
	DLock.RLock()
	count, ok := AllDealCountMap[epoch]
	DLock.RUnlock()

	if ok {
		return count, nil
	}

	filter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$eq", Value: epoch}}}}

	tableName := "DealProposal"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			count, err := col.CountDocuments(ctx, filter)
			if err != nil {
				return 0, err
			}

			DLock.Lock()
			AllDealCountMap[epoch] = count
			DLock.Unlock()

			return count, nil
		}
	}

	return 0, fmt.Errorf("no table DealProposal")
}
