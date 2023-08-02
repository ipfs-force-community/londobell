package aggregators

import (
	"context"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// formal db 每高度都更新，从formal获取即可, todo: 但会有延迟
func GetAddress(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAddress")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	//todo: 🍬？
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	addressRes, err := common.GetAddrs(ctx, req.Addr)
	if err != nil {
		if err != util.ErrNotFound {
			alog.Error(err)
		}

		util.ReturnOnErr(c, err)
		return
	}

	res.Data = addressRes
	c.JSON(http.StatusOK, res)
}
