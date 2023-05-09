package adapter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/filecoin-project/lotus/blockstore"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func CurrentSectorInitialPledge(c *gin.Context) {
	alog := log.With("method", "CurrentSectorInitialPledge")
	req := model.CurrentSectorInitialPledgeReq{}
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

	circ, err := api.StateVMCirculatingSupplyInternal(ctx, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pact, err := api.StateGetActor(ctx, power.Address, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))

	pst, err := power.Load(stor, pact)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	QualityAdjPowerSmoothed, err := pst.TotalPowerSmoothed()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ract, err := api.StateGetActor(ctx, reward.Address, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	rst, err := reward.Load(stor, ract)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	//thisEpochBaselinePower, err := rst.ThisEpochBaselinePower()
	//if err != nil {
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}

	//thisEpochRewardSmoothed, err := rst.ThisEpochRewardSmoothed()
	//if err != nil {
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}

	// 1TB: 1099511627776
	initPledge, err := rst.InitialPledgeForPower(big.MustFromString(req.QualityAdjPower), abi.NewTokenAmount(0), &QualityAdjPowerSmoothed, circ.FilCirculating)
	//initPledge := miner9.InitialPledgeForPower(big.MustFromString(req.QualityAdjPower), thisEpochBaselinePower, smoothing.FilterEstimate(thisEpochRewardSmoothed), smoothing.FilterEstimate(QualityAdjPowerSmoothed), circ.FilCirculating)
	circulatingf, err := strconv.ParseFloat(circ.FilCirculating.String(), 64)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalf, err := strconv.ParseFloat("2000000000000000000000000000", 64)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	resData := model.CurrentSectorInitialPledgeRes{
		CirculatingRate:            circulatingf / totalf,
		FilVested:                  circ.FilVested,
		FilMined:                   circ.FilMined,
		FilBurnt:                   circ.FilBurnt,
		FilLocked:                  circ.FilLocked,
		FilCirculating:             circ.FilCirculating,
		FilReserveDisbursed:        circ.FilReserveDisbursed,
		CurrentSectorInitialPledge: initPledge,
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
