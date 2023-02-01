package aggregators

import (
	"context"
	"fmt"
	"net/http"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
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

	formal := multiquery.DBStateManager.GetFormalCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(formal.Url())
	if !ok {
		return nil, fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "FinalHeight" {
			pipe, err := util.Parse(model.Ctx{}, string(finalHeightAggregator))
			if err != nil {
				return nil, err
			}
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return nil, err
			}

			err = cur.All(ctx, &finalHeightRes)
			if err != nil {
				return nil, err
			}

			return finalHeightRes, nil
		}
	}

	return nil, fmt.Errorf("no table FinalHeight")
}
