package adapter

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v8/miner"
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
		util.ReturnOnErr(c, alog, err)
		return
	}

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

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	resData := model.SectorPowerRes{}

	si, err := api.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(req.Sector), ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	size, err := si.SealProof.SectorSize()
	if err != nil {
		util.ReturnOnErr(c, alog, err)
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
