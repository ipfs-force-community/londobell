package chain

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetChainHead(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainHead")
	req := lotusCmdModel.ChainHeadReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := adapter.API.GetAppropriateAPI()
	head, err := api.ChainHead(ctx)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.ChainHeadRes{
		Head: head,
	}

	c.JSON(http.StatusOK, res)
}
