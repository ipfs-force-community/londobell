package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetFinalHeight(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetFinalHeight")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var finalHeightRes []model.FinalHeightRes
	pipe, err := Parse(model.Ctx{}, string(finalHeightAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}
	cur, err := mongoutil.FinalHeightCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &finalHeightRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = finalHeightRes
	c.JSON(http.StatusOK, res)
}
