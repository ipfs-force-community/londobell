package aggregators

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"

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

	finalHeight, err := GetFinalHeightForFormalDB(ctx)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = []model.FinalHeightRes{model.FinalHeightRes{
		Epoch: finalHeight,
	}}
	c.JSON(http.StatusOK, res)
}

func GetFinalHeightForFormalDB(ctx context.Context) (abi.ChainEpoch, error) {
	cols, ok := multiquery.DBStateManager.GetDBCollections(multiquery.DBStateManager.GetFormalCfg().Url())
	if !ok {
		return 0, fmt.Errorf("url %v not found in DBCollectionsMap", multiquery.DBStateManager.GetFormalCfg().Url())
	}

	return multiquery.GetFinalHeight(ctx, cols)
}
