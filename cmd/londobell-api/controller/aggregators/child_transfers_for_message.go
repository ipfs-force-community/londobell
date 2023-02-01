package aggregators

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetChildTransfersForMessage(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetChildTransfersForMessage")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var childTransfersForMessageRes []model.ChildTransfersForMessageRes
	pipe, err := Parse(model.Ctx{Cid: req.Cid}, string(childTransfersForMessageAggregator))
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

	err = cur.All(ctx, &childTransfersForMessageRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// search from the temporary repository if not found
	if len(childTransfersForMessageRes) == 0 {
		tmpPipe, err := Parse(model.Ctx{Cid: req.Cid}, string(childTransfersForMessageAggregator))
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

		err = tmpCur.All(ctx, &childTransfersForMessageRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = childTransfersForMessageRes
	c.JSON(http.StatusOK, res)
}
