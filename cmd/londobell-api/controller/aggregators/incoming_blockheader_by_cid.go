package aggregators

import (
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetIncomingBlockHeaderByCid(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetIncomingBlockHeaderByCid")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pipe, err := util.Parse(model.Ctx{Cid: req.Cid}, string(common.BlockHeaderByCidAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var blockHeaderRes []model.BlockHeader

	blockHeaderRes, err = GetIncomingBlockForFormalDB(ctx, pipe)

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = blockHeaderRes
	c.JSON(http.StatusOK, res)
}
