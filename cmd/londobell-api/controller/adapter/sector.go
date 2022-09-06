package adapter

import (
	"context"
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

func GetSectorInfo(c *gin.Context) {
	alog := log.With("method", "GetSectorInfo")
	req := model.SectorReq{}
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

	resDatas := make([]model.SectorRes, 0)

	sectors, err := api.StateMinerSectors(ctx, maddr, nil, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	for _, info := range sectors {
		resData := model.SectorRes{}
		resData.Miner = maddr
		resData.Date = common.CalcTimeByEpoch(uint64(info.Expiration))
		resData.SectorNumber = info.SectorNumber

		if info.SealProof >= 0 && info.SealProof <= 4 {
			resData.Version = "V1"
		} else {
			resData.Version = "V1_1"
		}

		resData.Size, err = info.SealProof.SectorSize()
		if err != nil {
			util.ReturnOnErr(c, alog, err)
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
