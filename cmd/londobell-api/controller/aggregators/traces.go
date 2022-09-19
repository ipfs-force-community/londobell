package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetTraces(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAddress")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var tracesRes []model.TraceRes

	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch}, string(tracesAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &tracesRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = tracesRes
	c.JSON(http.StatusOK, res)
}
