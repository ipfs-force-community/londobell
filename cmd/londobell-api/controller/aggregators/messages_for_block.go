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

func GetMessagesForBlock(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMessagesForBlock")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	skip, limit := req.Index*req.Limit, req.Limit
	if req.Index == 0 && req.Limit == 0 {
		skip = 0
		limit = math.MaxInt64
	}

	var messagesForBlockRes []model.TraceForMessageRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, Cid: req.Cid, Skip: skip, Limit: limit}, string(messagesForBlockAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.MessageBlockCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &messagesForBlockRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = messagesForBlockRes
	c.JSON(http.StatusOK, res)
}
