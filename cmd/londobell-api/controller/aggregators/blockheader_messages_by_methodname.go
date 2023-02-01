package aggregators

import (
	"context"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
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

	var blockHeaderMessagesByMethodNameRes []model.BlockHeaderMessagesByMethodNameRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, Cid: req.Cid, MethodName: req.MethodName}, string(countOfMessagesForBlockHeaderByMethodNameAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.MessageBlockCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &blockHeaderMessagesByMethodNameRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if len(blockHeaderMessagesByMethodNameRes) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	blockHeaderMessagesByMethodName := blockHeaderMessagesByMethodNameRes[0]

	skip, limit := req.Index*req.Limit, req.Limit
	if req.Index == 0 && req.Limit == 0 {
		skip = 0
		limit = math.MaxInt64
	}

	var messagesForBlockByMethodNameRes []model.TraceForMessageRes
	pipe, err = Parse(model.Ctx{StartEpoch: req.StartEpoch, Cid: req.Cid, MethodName: req.MethodName, Skip: skip, Limit: limit}, string(blockHeaderMessagesByMethodNameAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err = mongoutil.MessageBlockCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &messagesForBlockByMethodNameRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	blockHeaderMessagesByMethodName.Messages = messagesForBlockByMethodNameRes
	res.Data = blockHeaderMessagesByMethodName
	c.JSON(http.StatusOK, res)
}
