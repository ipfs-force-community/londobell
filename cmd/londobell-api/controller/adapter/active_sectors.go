package adapter

import (
	"context"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"

	"github.com/filecoin-project/go-address"
	"github.com/gin-gonic/gin"

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

// ComputeQAPower 按 NV29（Solstice / FIP-0118）口径汇总一个 miner 活跃扇区的 QA 算力与
// VDC/DC/CC 分桶。口径实现见同包 qa_split.go（与 filscan_backend/pkg/londobell/qa_split.go 同源）。
//
// 老实现的两处问题（都已修）：
//  1. 用 sminer(v11).QAPowerForSector：duration 取 Expiration-Activation，且不认
//     FULL_QA_POWER 标志 —— NV29 之后新扇区（10x）被算成 1x，续期扇区 duration 偏大；
//  2. adjPower 与三桶一路 .Int64() 截断（超过 9.22e18 直接溢出成负数/垃圾值），
//     改成全程 big.Int / decimal。
func ComputeQAPower(sectorInfos []*miner.SectorOnChainInfo, sectorSize abi.SectorSize) (model.QAPowerRes, []model.SectorOnChainInfo) {
	totalVDCPower, totalDCPower, totalCCPower := decimal.Zero, decimal.Zero, decimal.Zero
	sectorExpirations := make([]model.SectorOnChainInfo, 0, len(sectorInfos))
	for _, sectorInfo := range sectorInfos {
		// 逐扇区切 QA / VDC / DC / CC（三桶之和恒等于该扇区 QA 算力）。
		_, vdc, dc, cc, full := QASplit(
			uint64(sectorSize),
			uint64(sectorInfo.Flags),
			int64(sectorInfo.PowerBaseEpoch),
			int64(sectorInfo.Activation),
			int64(sectorInfo.Expiration),
			sectorInfo.DealWeight.Int,
			sectorInfo.VerifiedDealWeight.Int,
		)

		sectorExpirations = append(sectorExpirations, model.SectorOnChainInfo{
			Expiration:         sectorInfo.Expiration,
			Activation:         sectorInfo.Activation,
			DealWeight:         sectorInfo.DealWeight,
			VerifiedDealWeight: sectorInfo.VerifiedDealWeight,
			InitialPledge:      sectorInfo.InitialPledge,
			// NV29 新增：Flags / PowerBaseEpoch 原样带出，FullQaPower 与上面的分桶口径一致
			// （等价于 lotus miner.SectorIsFullQaPower，见 qa_split_test.go 里的对齐测试）。
			Flags:          uint64(sectorInfo.Flags),
			PowerBaseEpoch: sectorInfo.PowerBaseEpoch,
			FullQaPower:    full,
		})

		totalVDCPower = totalVDCPower.Add(decimal.NewFromBigInt(vdc, 0))
		totalDCPower = totalDCPower.Add(decimal.NewFromBigInt(dc, 0))
		totalCCPower = totalCCPower.Add(decimal.NewFromBigInt(cc, 0))
	}

	return model.QAPowerRes{VDCPower: totalVDCPower, DCPower: totalDCPower, CCPower: totalCCPower}, sectorExpirations
}
