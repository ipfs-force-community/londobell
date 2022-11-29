package state

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateListActors(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateListActors")
	req := lotusCmdModel.StateListActorsReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	actors, err := api.StateListActors(ctx, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateListActorsRes{
		Epoch:  ts.Height(),
		Actors: actors,
	}

	c.JSON(http.StatusOK, res)
}
