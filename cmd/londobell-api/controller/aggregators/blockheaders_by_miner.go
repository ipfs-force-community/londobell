package aggregators

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetBlockHeadersByMiner(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlockHeadersByMiner")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var blockHeadersByMinerRes []model.BlockHeader
	// todo: 默认返回所有数据，防止恶意攻击
	limit := req.Limit
	if req.Index == 0 && req.Limit == 0 {
		limit = math.MaxInt64
	}

	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, EndEpoch: req.EndEpoch, Addr: req.Addr, Skip: req.Index * req.Limit, Limit: limit}, string(blockHeadersByMinerAggregator))
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

	err = cur.All(ctx, &blockHeadersByMinerRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.BlockHeaderRes{TotalCount: int64(len(blockHeadersByMinerRes)), BlockHeaders: blockHeadersByMinerRes}
	c.JSON(http.StatusOK, res)
}
