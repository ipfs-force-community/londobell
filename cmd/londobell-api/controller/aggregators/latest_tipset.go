package aggregators

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetLatestTipSet(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetLatestTipSet")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	latestTipSetRes, err := getLatestTipSet(ctx)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	res.Data = latestTipSetRes
	c.JSON(http.StatusOK, res)
}

func getLatestTipSet(ctx context.Context) ([]model.TipSetRes, error) {
	var latestTipSetRes []model.TipSetRes
	tmp := multiquery.DBStateManager.GetTmpCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(tmp.Url())
	if !ok {
		return latestTipSetRes, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
	}

	pipe, err := util.Parse(model.Ctx{}, string(common.LatestTipSetAggregator))
	if err != nil {
		return latestTipSetRes, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "Tipset" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return latestTipSetRes, err
			}

			err = cur.All(ctx, &latestTipSetRes)
			if err != nil {

				return latestTipSetRes, err
			}
		}
	}
	return latestTipSetRes, nil
}
