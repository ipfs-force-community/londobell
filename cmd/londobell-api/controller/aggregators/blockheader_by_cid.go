package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetBlockHeaderByCid(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlockHeaderByCid")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var blockHeaderRes []model.BlockHeader
	pipe, err := Parse(model.Ctx{Cid: req.Cid}, string(blockHeaderByCidAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.BlockHeaderCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &blockHeaderRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// get near-height data from the temporary repository
	if len(blockHeaderRes) == 0 {
		tmpPipe, err := Parse(model.Ctx{Cid: req.Cid}, string(blockHeaderByCidAggregator))
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		tmpCur, err := mongoutil.TmpBlockHeaderCol.Aggregate(ctx, tmpPipe)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = tmpCur.All(ctx, &blockHeaderRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = blockHeaderRes
	c.JSON(http.StatusOK, res)
}
