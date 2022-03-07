package controller

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v8/miner"
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
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	size, err := si.SealProof.SectorSize()
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] SectorSize err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	qualityAdjPower := miner.QAPowerForWeight(size, si.Expiration-si.Activation, si.DealWeight, si.VerifiedDealWeight)
	resData.Miner = maddr
	resData.Epoch = ts.Height()
	resData.Sector = req.Sector
	resData.QualityAdjPower = qualityAdjPower

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
