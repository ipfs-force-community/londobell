package state

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateGetActor(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateGetActor")
	req := lotusCmdModel.StateGetActorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.Addr == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass address of actor to get"))
		return
	}

	addr, err := address.NewFromString(req.Addr)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var ts *types.TipSet
	api := adapter.API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	actor, err := api.StateGetActor(ctx, addr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	strtype := builtin.ActorNameByCode(actor.Code)

	res.Data = lotusCmdModel.StateGetActorRes{
		Epoch: ts.Height(),
		Addr:  addr,
		Actor: actor,
		Type:  strtype,
	}

	c.JSON(http.StatusOK, res)
}
