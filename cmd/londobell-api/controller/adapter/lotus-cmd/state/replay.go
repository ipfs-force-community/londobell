package state

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateReplay(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateReplay")
	req := lotusCmdModel.StateReplayReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.MessageCid == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("message cid was invalid: %s", err))
		return
	}

	mcid, err := cid.Decode(req.MessageCid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	api := adapter.API.GetAppropriateAPI()

	// todo: replay may spend too long time
	result, err := api.StateReplay(ctx, types.EmptyTSK, mcid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateReplayRes{
		MessageCid: mcid,
		Result:     result,
	}

	c.JSON(http.StatusOK, res)
}
