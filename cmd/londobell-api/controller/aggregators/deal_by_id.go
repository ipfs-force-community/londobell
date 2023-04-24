package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetDealByID(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetDealByID")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var dealRes []model.Deal
	pipe, err := Parse(model.Ctx{ID: req.ID}, string(dealByIDAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.DealProposalCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &dealRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = dealRes
	c.JSON(http.StatusOK, res)
}
