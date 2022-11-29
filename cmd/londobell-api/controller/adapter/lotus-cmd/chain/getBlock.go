package chain

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

func GetChainGetBlock(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainGetBlock")
	req := lotusCmdModel.ChainGetBlockReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.BlockCid == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass cid of block to print"))
		return
	}

	bcid, err := cid.Decode(req.BlockCid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	api := adapter.API.GetAppropriateAPI()

	blk, err := api.ChainGetBlock(ctx, bcid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	msgs, err := api.ChainGetBlockMessages(ctx, bcid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	pmsgs, err := api.ChainGetParentMessages(ctx, bcid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	recpts, err := api.ChainGetParentReceipts(ctx, bcid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.ChainGetBlockRes{
		BlockCid:       bcid,
		BlockHeader:    *blk,
		BlsMessages:    msgs.BlsMessages,
		SecpkMessages:  msgs.SecpkMessages,
		ParentReceipts: recpts,
		ParentMessages: apiMsgCids(pmsgs),
	}

	c.JSON(http.StatusOK, res)
}
