package chain

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs/go-cid"
)

func GetChainGetMessage(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainGetMessage")
	req := lotusCmdModel.ChainGetMessageReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.MessageCid == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass cid of a message to get"))
		return
	}

	cid, err := cid.Decode(req.MessageCid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	api := adapter.API.GetAppropriateAPI()

	mb, err := api.ChainReadObj(ctx, cid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var i interface{}
	m, err := types.DecodeMessage(mb)
	if err != nil {
		sm, err := types.DecodeSignedMessage(mb)
		if err != nil {
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
		}
		i = sm
	} else {
		i = m
	}

	res.Data = lotusCmdModel.ChainGetMessageRes{
		MessageCid: cid,
		Message:    i,
	}

	c.JSON(http.StatusOK, res)
}
