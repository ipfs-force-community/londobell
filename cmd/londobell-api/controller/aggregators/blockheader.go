package aggregators

import (
	"context"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetBlockHeader(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlockHeader")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var blockHeaderRes []model.BlockHeader
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch}, string(blockHeaderAggregator))
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
	sort.Slice(blockHeaderRes, func(i, j int) bool {
		return blockHeaderRes[i].Epoch > blockHeaderRes[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(blockHeaderRes) > 0 {
		tmpStartEpoch = int64(blockHeaderRes[0].Epoch) + 1
	}

	var tmpBlockHeaderRes []model.BlockHeader
	tmpPipe, err := Parse(model.Ctx{StartEpoch: tmpStartEpoch, EndEpoch: req.EndEpoch}, string(blockHeaderAggregator))
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

	err = tmpCur.All(ctx, &tmpBlockHeaderRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	blockHeaderRes = append(blockHeaderRes, tmpBlockHeaderRes...)

	sort.Slice(blockHeaderRes, func(i, j int) bool {
		return blockHeaderRes[i].Epoch > blockHeaderRes[j].Epoch
	})

	// paging
	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.BlockHeaderRes{TotalCount: int64(len(blockHeaderRes)), BlockHeaders: blockHeaderRes}
		c.JSON(http.StatusOK, res)
		return
	}

	if req.Index*req.Limit >= int64(len(blockHeaderRes)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(blockHeaderRes)) {
		res.Data = model.BlockHeaderRes{TotalCount: int64(len(blockHeaderRes[req.Index*req.Limit:])), BlockHeaders: blockHeaderRes[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.BlockHeaderRes{TotalCount: int64(len(blockHeaderRes[req.Index*req.Limit : (req.Index+1)*req.Limit])), BlockHeaders: blockHeaderRes[req.Index*req.Limit : (req.Index+1)*req.Limit]}

	c.JSON(http.StatusOK, res)
}
