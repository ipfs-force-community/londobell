package controller

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetEpochInfo(c *gin.Context) {
	req := model.EpochReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("[GetEpochInfo] bind json EpochReq err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet

	if req.Epoch == 0 {
		ts, err = API.ChainHead(ctx)
		if err != nil {
			log.Errorf("[GetEpochInfo] api ChainHead err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			log.Errorf("[GetEpochInfo] api ChainGetTipSetByHeight err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	var winCount int64
	for _, b := range ts.Blocks() {
		winCount += b.ElectionProof.WinCount
	}

	pact, err := API.StateGetActor(ctx, power.Address, ts.Key())
	if err != nil {
		log.Errorf("[GetEpochInfo] api StateGetActor err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(API))

	pst, err := power.Load(stor, pact)
	if err != nil {
		log.Errorf("[GetEpochInfo] load pst err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	pc, err := pst.TotalPower()
	if err != nil {
		log.Errorf("[GetEpochInfo] get TotalPower err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	ract, err := API.StateGetActor(ctx, reward.Address, ts.Key())
	if err != nil {
		log.Errorf("[GetEpochInfo] api StateGetActor err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	rst, err := reward.Load(stor, ract)
	if err != nil {
		log.Errorf("[GetEpochInfo] load rst err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	currentTotalStoragePowerReward, err := rst.TotalStoragePowerReward()
	if err != nil {
		log.Errorf("[GetEpochInfo] get TotalStoragePowerReward err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	parentTs, err := API.ChainGetTipSet(ctx, types.NewTipSetKey(ts.Blocks()[0].Parents...))
	if err != nil {
		log.Errorf("[GetEpochInfo] api ChainGetTipSet err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	parentRoot := parentTs.ParentState()
	parentTree, err := state.LoadStateTree(stor, parentRoot)
	if err != nil {
		log.Errorf("[GetEpochInfo] LoadStateTree err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	pract, err := parentTree.GetActor(reward.Address)
	if err != nil {
		log.Errorf("[GetEpochInfo] GetActor err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	prst, err := reward.Load(stor, pract)
	if err != nil {
		log.Errorf("[GetEpochInfo] load prst err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	parentTotalStoragePowerReward, err := prst.TotalStoragePowerReward()
	if err != nil {
		log.Errorf("[GetEpochInfo] TotalStoragePowerReward err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	netRewards := big.Sub(currentTotalStoragePowerReward, parentTotalStoragePowerReward)

	resData := model.EpochRes{
		Cids:            ts.Cids(),
		Parents:         ts.Parents(),
		Epoch:           ts.Height(),
		BlockTime:       CalcTimeByEpoch(uint64(ts.Height())),
		BlockCount:      len(ts.Blocks()),
		WinCount:        winCount,
		NetPower:        pc.RawBytePower,
		NetQualityPower: pc.QualityAdjPower,
		NetRewards:      netRewards,
		BaseFee:         ts.Blocks()[0].ParentBaseFee,
		Source:          "api",
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
