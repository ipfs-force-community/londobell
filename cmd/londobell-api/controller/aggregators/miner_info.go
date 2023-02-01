package aggregators

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetMinerInfo(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMinerInfo")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var minerInfoRes []model.MinersInfoRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, Addr: req.Addr}, string(minerInfoAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.MinerFundsCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &minerInfoRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = minerInfoRes
	c.JSON(http.StatusOK, res)
}
