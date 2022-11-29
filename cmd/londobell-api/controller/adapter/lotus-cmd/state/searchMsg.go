package state

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateSearchMsg(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateSearchMsg")
	req := lotusCmdModel.StateSearchMsgReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.MessageCid == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must specify message cid to search for"))
		return
	}

	msg, err := cid.Decode(req.MessageCid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	api := adapter.API.GetAppropriateAPI()
	mw, err := api.StateSearchMsg(ctx, msg)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if mw == nil {
		util.ReturnOnErr(c, alog, fmt.Errorf("failed to find message: %s", msg))
		return
	}

	m, err := api.ChainGetMessage(ctx, msg)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateSearchMsgRes{
		MessageCid: msg,
		Message:    m,
	}

	c.JSON(http.StatusOK, res)
}
