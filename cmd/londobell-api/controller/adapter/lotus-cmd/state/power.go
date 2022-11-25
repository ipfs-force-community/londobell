package state

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/lotus/chain/actors/builtin"

	"github.com/filecoin-project/go-address"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	lapi "github.com/filecoin-project/lotus/api"
	lpower "github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStatePower(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStatePower")
	req := lotusCmdModel.StatePowerReq{}
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

	var (
		maddr address.Address
		power *lapi.MinerPower
		tp    lpower.Claim
		mp    lpower.Claim
	)

	if req.Miner != "" {
		maddr, err = address.NewFromString(req.Miner)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		ma, err := api.StateGetActor(ctx, maddr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		if !builtin.IsStorageMinerActor(ma.Code) {
			util.ReturnOnErr(c, alog, fmt.Errorf("provided address does not correspond to a miner actor"))
			return
		}
	}

	power, err = api.StateMinerPower(ctx, maddr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	tp = power.TotalPower
	mp = power.MinerPower

	res.Data = lotusCmdModel.StatePowerRes{
		Miner:      maddr,
		Epoch:      ts.Height(),
		MinerPower: mp,
		TotalPower: tp,
	}

	c.JSON(http.StatusOK, res)
}
