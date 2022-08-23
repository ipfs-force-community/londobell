package adapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
)

func GetBatchMinersInfo(c *gin.Context) {
	alog := log.With("method", "GetBatchMinersInfo")
	req := model.BatchMinersReq{}
	res := model.CommonRes{Code: model.Success}
	batchRes := model.BatchMinersRes{}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	fmt.Printf("BatchMinersReq: %v\n", req)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	api := API.GetAppropriateAPI()
	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	}

	for _, Miner := range req.Miners {
		maddr, err := address.NewFromString(Miner.Miner)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		mi, err := api.StateMinerInfo(ctx, maddr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		power, err := api.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		mact, err := api.StateGetActor(ctx, maddr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		if !builtin.IsStorageMinerActor(mact.Code) {
			util.ReturnOnErr(c, alog, fmt.Errorf("provided address does not correspond to a miner actor"))
			return
		}

		availableBalance, err := api.StateMinerAvailableBalance(ctx, maddr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))
		mas, err := miner.Load(stor, mact)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		lockedFunds, err := mas.LockedFunds()
		if err != nil {
			util.ReturnOnErr(c, alog, err)
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
			util.ReturnOnErr(c, alog, err)
			return
		}

		batchRes.MinersRes = append(batchRes.MinersRes, *resData)
	}

	res.Data = batchRes
	c.JSON(http.StatusOK, res)
}
