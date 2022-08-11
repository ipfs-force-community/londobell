package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetBlock(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlock")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var blockRes []model.BlockRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch}, string(blockAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &blockRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = blockRes
	c.JSON(http.StatusOK, res)
}
