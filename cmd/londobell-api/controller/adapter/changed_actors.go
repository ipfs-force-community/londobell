package adapter

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateChaingedActors(c *gin.Context) {
	alog := log.With("method", "GetStateChaingedActors")
	req := model.EpochReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet

	api := fullnode.API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	parent, err := api.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	old := parent.ParentState()
	new := ts.ParentState()
	changedActors, err := api.StateChangedActors(ctx, old, new)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = changedActors
	c.JSON(http.StatusOK, res)
}
