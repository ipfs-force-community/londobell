package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
)

func GetBatchMinersInfo(c *gin.Context) {
	req := model.BatchMinersReq{}
	res := model.CommonRes{Code: model.Success}
	batchRes := model.BatchMinersRes{}
	err := c.BindJSON(&req)
	if err != nil {
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	fmt.Printf("BatchMinersReq: %v\n", req)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	if req.Epoch == 0 {
		ts, err = API.ChainHead(ctx)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	for _, Miner := range req.Miners {
		maddr, err := address.NewFromString(Miner.Miner)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		mi, err := API.StateMinerInfo(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		power, err := API.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		mact, err := API.StateGetActor(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		if !builtin.IsStorageMinerActor(mact.Code) {
			res.Code = model.Fail
			res.Msg = "provided address does not correspond to a miner actor"
			c.JSON(http.StatusOK, res)
			return
		}

		availableBalance, err := API.StateMinerAvailableBalance(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(API))
		mas, err := miner.Load(stor, mact)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		lockedFunds, err := mas.LockedFunds()
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		resData := &model.MinerRes{}

		resData.Epoch = ts.Height()
		resData.Miner = maddr
		resData.Owner = mi.Owner
		resData.Worker = mi.Worker
		resData.Controllers = mi.ControlAddresses
		resData.SectorSize = mi.SectorSize
		resData.Power = power.MinerPower.RawBytePower
		resData.QualityPower = power.MinerPower.QualityAdjPower
		resData.Balance = mact.Balance
		resData.AvailableBalance = availableBalance
		resData.VestingFunds = lockedFunds.VestingFunds
		resData.LockedFunds = lockedFunds.PreCommitDeposits
		resData.InitialPledgeRequirement = lockedFunds.InitialPledgeRequirement

		err = getMinerResByCode(ctx, mact, stor, resData)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
		}

		batchRes.MinersRes = append(batchRes.MinersRes, *resData)
	}

	res.Data = batchRes
	c.JSON(http.StatusOK, res)
}
