package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	pool_monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetBlockHeadersByMiner(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlockHeadersByMiner")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	req.Addr, err = common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetTotalCountForMinedMsgsMap(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.MinedStates
	}

	var blockHeadersByMiner []model.BlockHeader
	// multi dbs query
	{
		multiResult, err := multiquery.MultiBiSearch(ctx, req.Index*req.Limit, req.Limit, countUtils, pool_monitor.GetBlockHeadersByMinerNoSkipAggregator(),
			pool_monitor.GetMinedCountForMinersAggregator(), req, "BlockHeader", multiquery.MinedStates)
		//multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.MinedStates, countUtils, blockHeadersByMinerAggregator, req, "BlockHeader")
		if err != nil {
			//alog.Error(err)
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

		err = json.Unmarshal(rawByte, &blockHeadersByMiner)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.BlockHeaderRes{TotalCount: totalCount, BlockHeaders: blockHeadersByMiner}
	c.JSON(http.StatusOK, res)
}
