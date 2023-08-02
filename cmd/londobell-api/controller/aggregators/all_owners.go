package aggregators

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// MinerFunds 每高度都会更新，只查formal即可
func GetAllOwners(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAllOwners")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	formal := multiquery.DBStateManager.GetFormalCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(formal.Url())
	if !ok {
		alog.Error(fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url()))
		util.ReturnOnErr(c, err)
		return
	}

	latestEpoch, err := GetLatestEpoch(ctx, cols, "MinerFunds")
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var allOwnersRes []model.AllOwnersRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(latestEpoch)}, string(common.AllOwnersAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "MinerFunds" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = cur.All(ctx, &allOwnersRes)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			res.Data = allOwnersRes
			c.JSON(http.StatusOK, res)

			return
		}
	}

	c.JSON(http.StatusOK, res)
}
