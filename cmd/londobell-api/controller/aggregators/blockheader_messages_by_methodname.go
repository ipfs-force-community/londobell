package aggregators

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetBlockHeaderMessagesByMethodName(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlockHeaderMessagesByMethodName")
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

	if req.Index == 0 && req.Limit == 0 {
		req.Limit = math.MaxInt64
	}

	var (
		blockHeaderMessagesByMethodNameRes model.BlockHeaderMessagesByMethodNameRes
		messagesForBlockByMethodNameRes    []model.TraceForMessageRes
	)

	// multi dbs query
	{
		totalCountResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.StartEpoch+1, countUtils, countOfMessagesForBlockHeaderByMethodNameAggregator, req, "BlockMessage")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(totalCountResult) == 0 {
			c.JSON(http.StatusOK, res)
			return
		}

		countRaw := totalCountResult[0]
		countRawByte, err := json.Marshal(countRaw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(countRawByte, &blockHeaderMessagesByMethodNameRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.StartEpoch+1, countUtils, blockHeaderMessagesByMethodNameAggregator, req, "MessageBlock")
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

		err = json.Unmarshal(rawByte, &messagesForBlockByMethodNameRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		blockHeaderMessagesByMethodNameRes.Messages = messagesForBlockByMethodNameRes
	}

	res.Data = blockHeaderMessagesByMethodNameRes
	c.JSON(http.StatusOK, res)
}
