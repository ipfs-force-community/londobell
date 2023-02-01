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
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	finalHeightRes, err := GetFinalHeightForFormalDB(ctx)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = finalHeightRes
	c.JSON(http.StatusOK, res)
}

func GetFinalHeightForFormalDB(ctx context.Context) ([]model.FinalHeightRes, error) {
	var finalHeightRes []model.FinalHeightRes
	pipe, err := Parse(model.Ctx{}, string(finalHeightAggregator))
	if err != nil {
		return nil, err
	}
	cur, err := mongoutil.FinalHeightCol.Aggregate(ctx, pipe)
	if err != nil {
		return nil, err
	}

	err = cur.All(ctx, &finalHeightRes)
	if err != nil {
		return nil, err
	}

	return finalHeightRes, nil
}
