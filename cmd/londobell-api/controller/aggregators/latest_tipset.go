package aggregators

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

	tmp := multiquery.DBStateManager.GetTmpCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(tmp.Url())
	if !ok {
		alog.Error(fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url()))
		util.ReturnOnErr(c, err)
		return
	}

	var latestTipSetRes []model.TipSetRes
	pipe, err := util.Parse(model.Ctx{}, string(latestTipSetAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "Tipset" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = cur.All(ctx, &latestTipSetRes)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			// todo: no data, then search from formal、cold; alert if outdated
			res.Data = latestTipSetRes
			c.JSON(http.StatusOK, res)

			return
		}
	}

	c.JSON(http.StatusOK, res)
}
