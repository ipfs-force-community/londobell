package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetRichList(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetRichList")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var richList []model.Rich
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch}, string(richListAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.ActorBalanceCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &richList)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// get near-height data from the temporary repository
	if len(richList) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.RichListRes{TotalCount: int64(len(richList)), RichList: richList}
		c.JSON(http.StatusOK, res)
		return
	}
	// paging
	if req.Index*req.Limit >= int64(len(richList)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(richList)) {
		res.Data = model.RichListRes{TotalCount: 1000, RichList: richList[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.RichListRes{TotalCount: 1000, RichList: richList[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
