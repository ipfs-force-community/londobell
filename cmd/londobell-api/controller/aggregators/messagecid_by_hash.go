package aggregators

import (
	"net/http"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetMessageCidByHash(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMessageCidByHash")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cid, err := GetCidFromEthHash(ctx, req.Cid)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.MessageCidByHashRes{Cid: cid}

	c.JSON(http.StatusOK, res)
}
