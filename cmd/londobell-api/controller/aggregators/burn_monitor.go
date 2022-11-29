package aggregators

import (
	"context"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetBurnMonitor(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBurnMonitor")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var burnMonitorRes []model.BurnMonitorRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch, Addr: req.Addr}, string(burnMonitorAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &burnMonitorRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = burnMonitorRes
	c.JSON(http.StatusOK, res)
}
