package controller

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetSectorInfo(c *gin.Context) {
	req := model.SectorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("[GetSectorInfo] bind json SectorReq err: %w", err)
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
			log.Errorf("[GetSectorInfo] api ChainHead err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			log.Errorf("[GetSectorInfo] api ChainGetTipSetByHeight err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		log.Errorf("[GetSectorInfo] address.NewFromString err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	resDatas := make([]model.SectorRes, 0)

	sectors, err := API.StateMinerSectors(ctx, maddr, nil, ts.Key())
	if err != nil {
		log.Errorf("[GetSectorInfo] api ChainGetTipSetByHeight err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	for _, info := range sectors {
		resData := model.SectorRes{}
		resData.Miner = maddr
		resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
		resData.SectorNumber = info.SectorNumber

		if info.SealProof >= 0 && info.SealProof <= 4 {
			resData.Version = "V1"
		} else {
			resData.Version = "V1_1"
		}

		resData.Size, err = info.SealProof.SectorSize()
		if err != nil {
			log.Errorf("[GetSectorInfo] miner0 SectorSize err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		resData.Activation = info.Activation
		resData.Expiration = info.Expiration
		resData.Pledge = info.InitialPledge
		resData.DealWeight = info.DealWeight
		resData.VerifiedDealWeight = info.VerifiedDealWeight
		resData.ExpectedDayReward = info.ExpectedDayReward
		resData.ExpectedStoragePledge = info.ExpectedStoragePledge
		resData.ReplaceSectorAge = 0
		resData.ReplaceDayReward = big.NewInt(0)

		resDatas = append(resDatas, resData)
	}

	res.Data = resDatas
	c.JSON(http.StatusOK, res)
}
