package aggregators

import (
	"net/http"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetHashByMessageCid(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetHashByMessageCid")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	hash, err := GetEthHashByCid(ctx, req.Cid)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.HashByMessageCidRes{Hash: hash}

	c.JSON(http.StatusOK, res)
}
