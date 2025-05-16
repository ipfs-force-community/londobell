package adapter

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
		req.Epoch = int64(ts.Height())
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

	qualityAdjPowerSmoothed, err := pst.TotalPowerSmoothed()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pledgeCollateral, err := pst.TotalLocked()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var epochsSinceRampStart int64
	var rampDurationEpochs uint64
	if pst.RampStartEpoch() > 0 {
		epochsSinceRampStart = req.Epoch - pst.RampStartEpoch()
		rampDurationEpochs = pst.RampDurationEpochs()
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

	// default 1TB: 1099511627776
	if req.QualityAdjPower == "" {
		req.QualityAdjPower = "1099511627776"
	}

	initPledge, err := rst.InitialPledgeForPower(
		big.MustFromString(req.QualityAdjPower),
		pledgeCollateral,
		&qualityAdjPowerSmoothed,
		circ.FilCirculating,
		epochsSinceRampStart,
		rampDurationEpochs,
	)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

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
	dailyFee, err := dailyProofFee(circ.FilCirculating, big.MustFromString(req.QualityAdjPower))
	if err == nil {
		resData.DailyFee = dailyFee
	} else {
		alog.Error("dailyProofFee failed: ", err)
		resData.DailyFee = big.NewInt(0)
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}

const DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_NUM = 161817

var DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_DENOM = big.NewInt(0)

func init() {
	var err error
	DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_DENOM, err = big.FromString(
		strings.ReplaceAll("1_000_000_000_000_000_000_000_000_000_000", "_", ""))
	if err != nil {
		fmt.Printf("parse DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_DENOM failed: %v\n", err)
	}
}

// pub fn daily_proof_fee: https://github.com/filecoin-project/builtin-actors/blob/41bb0b28b479e75ea5d841fd216612acdbdb74c8/actors/miner/src/policy.rs#L214
func dailyProofFee(filCirculating abi.TokenAmount, qaPower abi.TokenAmount) (big.Int, error) {
	if DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_DENOM.IsZero() {
		return big.Int{}, fmt.Errorf("DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_DENOM is zero")
	}

	val := big.NewInt(0).Mul(filCirculating.Int,
		big.NewInt(DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_NUM).Int)
	val = big.NewInt(0).Mul(val, qaPower.Int)

	ret := big.NewInt(0).Div(val, DAILY_FEE_CIRCULATING_SUPPLY_QAP_MULTIPLIER_DENOM.Int)
	return big.Int{Int: ret}, nil
}
