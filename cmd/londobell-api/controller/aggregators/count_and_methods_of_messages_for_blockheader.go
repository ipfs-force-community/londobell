package aggregators

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetCountAndMethodsOfMessagesForBlockHeader(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetCountAndMethodsOfMessagesForBlockHeader")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var countAndMethodsForBlockHeaderRes []model.CountAndMethodsForBlockHeader
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, Cid: req.Cid}, string(countAndMethodNameOfMessagesForBlockHeaderAggregator))
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

	err = cur.All(ctx, &countAndMethodsForBlockHeaderRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if len(countAndMethodsForBlockHeaderRes) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = countAndMethodsForBlockHeaderRes[0]
	c.JSON(http.StatusOK, res)
}
