package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetMinedByMinerForRange(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMinedByMinerForRange")
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

	req.Addr, err = common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var minerByMinerForRangeRes []model.MinedByMinerForRangeRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, common2.MinedByMinerRangeAggregator, req, "BlockHeader")
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

		err = json.Unmarshal(rawByte, &minerByMinerForRangeRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = minerByMinerForRangeRes
	c.JSON(http.StatusOK, res)
}
