package aggregators

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetIncomingBlockHeader(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetIncomingBlockHeader")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pipe, err := util.Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch}, string(common2.BlockHeaderAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	blockHeaderRes, err := GetIncomingBlockForFormalDB(ctx, pipe)

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	res.Data = blockHeaderRes
	c.JSON(http.StatusOK, res)
}

func GetIncomingBlockForFormalDB(ctx context.Context, pipe interface{}) ([]model.BlockHeader, error) {
	cols, ok := multiquery.DBStateManager.GetDBCollections(multiquery.DBStateManager.GetFormalCfg().Url())
	if !ok {
		return nil, fmt.Errorf("url %v not found in DBCollectionsMap", multiquery.DBStateManager.GetFormalCfg().Url())
	}

	return multiquery.GetIncomingBlock(ctx, pipe, cols)
}
