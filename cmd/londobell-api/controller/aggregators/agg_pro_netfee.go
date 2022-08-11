package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetAggProNetfee(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAggProNetfee")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var aggProNetfeeRes []model.AggProNetfeeRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch}, string(aggProNetfeeAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &aggProNetfeeRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = aggProNetfeeRes
	c.JSON(http.StatusOK, res)
}
