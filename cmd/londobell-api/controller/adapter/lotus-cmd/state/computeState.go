package state

import (
	"context"
	"net/http"

	lapi "github.com/filecoin-project/lotus/api"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateComputeState(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateComputeState")
	req := lotusCmdModel.StateComputeStateReq{}
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

	h := abi.ChainEpoch(req.VMHeight)
	if h == 0 {
		h = ts.Height()
	}

	var msgs []*types.Message
	if req.ApplyMpoolMessages {
		pmsgs, err := api.MpoolSelect(ctx, ts.Key(), 1)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		for _, sm := range pmsgs {
			msgs = append(msgs, &sm.Message)
		}
	}

	var stout *lapi.ComputeStateOutput
	o, err := api.StateCompute(ctx, h, msgs, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	stout = o

	// todo: return value
	res.Data = lotusCmdModel.StateComputeStateRes{
		Epoch:       h,
		StateOutPut: stout,
	}

	c.JSON(http.StatusOK, res)
}
