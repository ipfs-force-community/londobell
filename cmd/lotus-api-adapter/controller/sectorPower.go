package controller

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetSectorPowerInfo(c *gin.Context) {
	req := model.SectorPowerReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] bind json SectorPowerReq err: %w", err)
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
			log.Errorf("[GetSectorPowerInfo] api ChainHead err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			log.Errorf("[GetSectorPowerInfo] api ChainGetTipSetByHeight err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] address.NewFromString err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	resData := model.SectorPowerRes{}

	si, err := API.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(req.Sector), ts.Key())
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] API.StateSectorGetInfo err: %w", err)
	}

	resData.Miner = maddr
	resData.Date = CalcTimeByEpoch(uint64(si.Expiration))
	resData.SectorNumber = si.SectorNumber
	if si.SealProof >= 0 && si.SealProof <= 4 {
		resData.Version = "V1"
	} else {
		resData.Version = "V1_1"
	}
	resData.Size, err = si.SealProof.SectorSize()
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] SectorSize err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	resData.Activation = si.Activation
	resData.Expiration = si.Expiration
	resData.Pledge = si.InitialPledge
	resData.DealWeight = si.DealWeight
	resData.VerifiedDealWeight = si.VerifiedDealWeight
	resData.ExpectedDayReward = si.ExpectedDayReward
	resData.ExpectedStoragePledge = si.ExpectedStoragePledge
	resData.ReplaceSectorAge = si.ReplacedSectorAge
	resData.ReplaceDayReward = si.ReplacedDayReward

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
