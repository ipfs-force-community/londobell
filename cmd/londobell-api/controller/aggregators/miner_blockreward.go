package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetMinerBlockReward(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMinerBlockReward")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var minerBlockRewardRes []model.MinerBlockRewardRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch, Addr: req.Addr}, string(minerBlockrewardAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &minerBlockRewardRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = minerBlockRewardRes
	c.JSON(http.StatusOK, res)
}
