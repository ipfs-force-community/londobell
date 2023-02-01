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

func GetMinerBlockReward(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMinerBlockReward")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var minerBlockRewardRes []model.MinerBlockRewardRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch, Addr: req.Addr}, string(minerBlockrewardAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &minerBlockRewardRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// get near-height data from the temporary repository
	sort.Slice(minerBlockRewardRes, func(i, j int) bool {
		return minerBlockRewardRes[i].Epoch > minerBlockRewardRes[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(minerBlockRewardRes) > 0 {
		tmpStartEpoch = int64(minerBlockRewardRes[0].Epoch) + 1
	}

	var tmpMinerBlockRewardRes []model.MinerBlockRewardRes
	tmpPipe, err := Parse(model.Ctx{StartEpoch: tmpStartEpoch, EndEpoch: req.EndEpoch, Addr: req.Addr}, string(minerBlockrewardAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	tmpCur, err := mongoutil.TmpTraceCol.Aggregate(ctx, tmpPipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = tmpCur.All(ctx, &tmpMinerBlockRewardRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	minerBlockRewardRes = append(minerBlockRewardRes, tmpMinerBlockRewardRes...)

	// sort
	sort.Slice(minerBlockRewardRes, func(i, j int) bool {
		return minerBlockRewardRes[i].Epoch > minerBlockRewardRes[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = minerBlockRewardRes
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(minerBlockRewardRes)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(minerBlockRewardRes)) {
		res.Data = minerBlockRewardRes[req.Index*req.Limit:]
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = minerBlockRewardRes[req.Index*req.Limit : (req.Index+1)*req.Limit]
	c.JSON(http.StatusOK, res)
}
