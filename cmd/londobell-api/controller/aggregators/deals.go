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

func GetDeals(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetDeals")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var deals []model.Deal
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch}, string(dealsAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.DealProposalCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &deals)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// sort
	sort.Slice(deals, func(i, j int) bool {
		return deals[i].ID > deals[j].ID
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.DealsRes{TotalCount: int64(len(deals)), Deals: deals}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(deals)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(deals)) {
		res.Data = model.DealsRes{TotalCount: int64(len(deals[req.Index*req.Limit:])), Deals: deals[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.DealsRes{TotalCount: int64(len(deals[req.Index*req.Limit : (req.Index+1)*req.Limit])), Deals: deals[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
