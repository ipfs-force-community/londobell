package adapter

import (
	"context"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	miner19 "github.com/filecoin-project/go-state-types/builtin/v19/miner"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetSectorPowerInfo(c *gin.Context) {
	alog := log.With("method", "GetSectorPowerInfo")
	req := model.SectorPowerReq{}
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

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	resData := model.SectorPowerRes{}

	si, err := api.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(req.Sector), ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	size, err := si.SealProof.SectorSize()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// NV29（Solstice / FIP-0118）口径：QAPowerForSector 认 FULL_QA_POWER 标志（恒 10x），
	// QA 周期起点是 PowerBaseEpoch 而不是 Activation。老代码用的是 v8 的
	// QAPowerForWeight(size, Expiration-Activation, DealWeight, VerifiedDealWeight)，
	// 既漏了 FULL 标志，也会把续期扇区的 duration 算大。
	qualityAdjPower := miner19.QAPowerForSector(size, si)
	resData.Miner = maddr
	resData.Epoch = ts.Height()
	resData.Sector = req.Sector
	resData.QualityAdjPower = qualityAdjPower

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
