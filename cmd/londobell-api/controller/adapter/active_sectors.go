package adapter

import (
	"context"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/go-state-types/big"

	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"

	"github.com/filecoin-project/go-address"
	"github.com/gin-gonic/gin"

	sminer "github.com/filecoin-project/go-state-types/builtin/v11/miner"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetActiveSectors(c *gin.Context) {
	alog := log.With("method", "GetActiveSectors")
	req := model.ActorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := fullnode.API.GetAppropriateAPI()

	var ts *types.TipSet
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

	addr, err := address.NewFromString(req.ActorID)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// qapower
	activeSectorInfos, err := api.StateMinerActiveSectors(ctx, addr, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	minerInfo, err := api.StateMinerInfo(ctx, addr, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	qapowerRes, sectorExpirations := ComputeQAPower(activeSectorInfos, minerInfo.SectorSize)

	// 前65位是初始化代码，不存储
	res.Data = model.SectorInfoRes{SectorExpirationRes: model.SectorExpirationRes{SectorExpirations: sectorExpirations}, QAPowerRes: qapowerRes}
	c.JSON(http.StatusOK, res)
}

// VDCPower = VerifiedDealWeight*10 / (Expiration - Activation)
// DCPower = DealWeight / (Expiration - Activation)
// CCPower = ((Expiration - Activation) * sectorsize - VerifiedDealWeight - DealWeight) / (Expiration - Activation)
func ComputeQAPower(sectorInfos []*miner.SectorOnChainInfo, sectorSize abi.SectorSize) (model.QAPowerRes, []model.SectorOnChainInfo) {
	var totalVDCPower, totalDCPower, totalCCPower = decimal.NewFromInt(0), decimal.NewFromInt(0), decimal.NewFromInt(0)
	sectorExpirations := make([]model.SectorOnChainInfo, 0, len(sectorInfos))
	for _, sectorInfo := range sectorInfos {
		sectorExpirations = append(sectorExpirations, model.SectorOnChainInfo{
			Expiration:         sectorInfo.Expiration,
			Activation:         sectorInfo.Activation,
			DealWeight:         sectorInfo.DealWeight,
			VerifiedDealWeight: sectorInfo.VerifiedDealWeight,
			InitialPledge:      sectorInfo.InitialPledge,
		})

		duration := sectorInfo.Expiration - sectorInfo.Activation
		VDC := big.Mul(sectorInfo.VerifiedDealWeight, big.NewInt(10))
		DC := sectorInfo.DealWeight

		rawPower := big.Mul(big.NewInt(int64(duration)), big.NewInt(int64(sectorSize)))
		CC := big.Sub(big.Sub(rawPower, sectorInfo.VerifiedDealWeight), sectorInfo.DealWeight)

		info := &sminer.SectorOnChainInfo{
			SectorNumber:          sectorInfo.SectorNumber,
			SealProof:             sectorInfo.SealProof,
			SealedCID:             sectorInfo.SealedCID,
			DealIDs:               sectorInfo.DeprecatedDealIDs,
			Activation:            sectorInfo.Activation,
			Expiration:            sectorInfo.Expiration,
			DealWeight:            sectorInfo.DealWeight,
			VerifiedDealWeight:    sectorInfo.VerifiedDealWeight,
			InitialPledge:         sectorInfo.InitialPledge,
			ExpectedDayReward:     big.Zero(),
			ExpectedStoragePledge: big.Zero(),

			SectorKeyCID: sectorInfo.SectorKeyCID,
		}
		if sectorInfo.ExpectedDayReward != nil {
			info.ExpectedDayReward = *sectorInfo.ExpectedDayReward
		}
		if sectorInfo.ExpectedStoragePledge != nil {
			info.ExpectedStoragePledge = *sectorInfo.ExpectedStoragePledge
		}

		adjPowerDecimal := decimal.NewFromInt(sminer.QAPowerForSector(sectorSize, info).Int64())

		all := big.Add(CC, big.Add(VDC, DC))
		allDecimal := decimal.NewFromInt(all.Int64())
		VDCDecimal := decimal.NewFromInt(VDC.Int64())
		DCDecimal := decimal.NewFromInt(DC.Int64())
		CCDecimal := decimal.NewFromInt(CC.Int64())

		totalVDCPower = totalVDCPower.Add(adjPowerDecimal.Mul(VDCDecimal.Div(allDecimal)))
		totalDCPower = totalDCPower.Add(adjPowerDecimal.Mul(DCDecimal.Div(allDecimal)))
		totalCCPower = totalCCPower.Add(adjPowerDecimal.Mul(CCDecimal.Div(allDecimal)))
	}

	return model.QAPowerRes{VDCPower: totalVDCPower, DCPower: totalDCPower, CCPower: totalCCPower}, sectorExpirations
}
