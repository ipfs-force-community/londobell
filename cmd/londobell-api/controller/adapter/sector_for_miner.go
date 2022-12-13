package adapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetSectorForMinerInfo(c *gin.Context) {
	alog := log.With("method", "GetSectorForMinerInfo")
	req := model.SectorForMinerReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	api := API.GetAppropriateAPI()
	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	si, err := api.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(req.SectorNumber), ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if si == nil {
		util.ReturnOnErr(c, alog, fmt.Errorf("sector %d for miner %s not found", req.SectorNumber, maddr))
		return
	}

	resData := model.SectorRes{}
	resData.Miner = maddr
	resData.Date = common.CalcTimeByEpoch(uint64(si.Expiration))
	resData.SectorNumber = si.SectorNumber

	if si.SealProof >= 0 && si.SealProof <= 4 {
		resData.Version = "V1"
	} else {
		resData.Version = "V1_1"
	}

	resData.Size, err = si.SealProof.SectorSize()
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	resData.Activation = si.Activation
	resData.Expiration = si.Expiration
	resData.Pledge = si.InitialPledge
	resData.DealWeight = si.DealWeight
	resData.VerifiedDealWeight = si.VerifiedDealWeight
	resData.ExpectedDayReward = si.ExpectedDayReward
	resData.ExpectedStoragePledge = si.ExpectedStoragePledge
	resData.ReplaceSectorAge = 0
	resData.ReplaceDayReward = big.NewInt(0)

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
