package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	adt5 "github.com/filecoin-project/specs-actors/v5/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	summaryDaysV5 = []abi.ChainEpoch{1, 7, 14, 30}
	for d := (miner5.MinSectorExpiration / builtin5.EpochsInDay); d <= (miner5.MaxSectorExpirationExtension / builtin5.EpochsInDay); d += 180 {
		summaryDaysV5 = append(summaryDaysV5, abi.ChainEpoch(d))
	}

	schema.Register(
		schema.Model{
			Name: "miner-deal-sector",
			D:    &model.MinerDealSector{},
		},
	)
	mustRegisterRegularExtractor("MinerSectorSummaryV5", extractMinerSectorSummaryV5)
}

var summaryDaysV5 []abi.ChainEpoch

func extractMinerSectorSummaryV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner5.State) error {
	if ticks := ctx.Opts.StateRegular.MinerSectorSummaryTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	if st.Sectors.Equals(emptyMinerStateV5.Sectors) {
		return nil
	}

	daysMax := summaryDaysV5[len(summaryDaysV5)-1]
	summaries := make([]*model.MinerSectorSummaryRange, 0, len(summaryDaysV5)+1)
	summariesInDays := make([]*model.MinerSectorSummaryRange, 0, int(daysMax+1))

	var prevDays abi.ChainEpoch = 0
	for _, days := range summaryDaysV5 {
		current := &model.MinerSectorSummaryRange{
			LowerBound:              prevDays * builtin5.EpochsInDay,
			UpperBound:              days * builtin5.EpochsInDay,
			SectorCount:             0,
			DealCount:               0,
			V1SectorCount:           0,
			TotalDealWeight:         big.NewInt(0),
			TotalVerifiedDealWeight: big.NewInt(0),
			TotalInitialPledge:      abi.NewTokenAmount(0),
			TotalV1InitialPledge:    abi.NewTokenAmount(0),
		}

		summaries = append(summaries, current)
		for i := abi.ChainEpoch(0); i < (days - prevDays); i++ {
			summariesInDays = append(summariesInDays, current)
		}

		prevDays = days
	}

	last := &model.MinerSectorSummaryRange{
		LowerBound:              prevDays * builtin5.EpochsInDay,
		UpperBound:              -1,
		SectorCount:             0,
		V1SectorCount:           0,
		DealCount:               0,
		TotalDealWeight:         big.NewInt(0),
		TotalVerifiedDealWeight: big.NewInt(0),
		TotalInitialPledge:      abi.NewTokenAmount(0),
		TotalV1InitialPledge:    abi.NewTokenAmount(0),
	}

	summaries = append(summaries, last)
	summariesInDays = append(summariesInDays, last)

	sectors, err := miner5.LoadSectors(adt5.WrapStore(ctx.C, ctx.D.ActorStore(ctx.C)), st.Sectors)
	if err != nil {
		return fmt.Errorf("load sectors from adt store: %w", err)
	}

	var out miner5.SectorOnChainInfo
	minerCommittedCapacity := uint64(0)
	actStore := ctx.D.ActorStore(ctx.C)
	minfo, err := st.GetInfo(actStore)
	if err != nil {
		return fmt.Errorf("get miner info failed :%w", err)
	}
	sectorSize := minfo.SectorSize

	mds := []model.MinerDealSector{}
	err = sectors.ForEach(&out, func(n int64) error {
		if out.Expiration <= head.Epoch {
			return nil
		}

		remainDuration := out.Expiration - head.Epoch
		remainDays := remainDuration / builtin5.EpochsInDay

		idx := int(remainDays)
		target := summariesInDays[idx]
		target.SectorCount++
		target.DealCount += uint64(len(out.DealIDs))
		target.TotalDealWeight = big.Add(target.TotalDealWeight, out.DealWeight)
		target.TotalVerifiedDealWeight = big.Add(target.TotalVerifiedDealWeight, out.VerifiedDealWeight)
		target.TotalInitialPledge = big.Add(target.TotalInitialPledge, out.InitialPledge)

		if out.SealProof < abi.RegisteredSealProof_StackedDrg2KiBV1_1 {
			target.TotalV1InitialPledge = big.Add(target.TotalV1InitialPledge, out.InitialPledge)
			target.V1SectorCount += 1
		}

		if len(out.DealIDs) == 0 {
			minerCommittedCapacity += uint64(sectorSize)
		} else {
			mds = append(mds, model.MinerDealSector{
				ID:                 fmt.Sprintf("%s-%d-%d", head.Addr, head.Epoch, out.SectorNumber),
				Epoch:              head.Epoch,
				SectorNumber:       out.SectorNumber,
				SealProof:          out.SealProof,
				DealIDs:            out.DealIDs,
				DealWeight:         out.DealWeight,
				VerifiedDealWeight: out.VerifiedDealWeight,
				InitialPledge:      out.InitialPledge,
				QAPower:            miner5.QAPowerForSector(sectorSize, &out),
				Miner:              head.Addr,
			})
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk through all sectors: %w", err)
	}

	id, err := GenRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("gen regular id: %w", err)
	}

	nonEmpty := make([]*model.MinerSectorSummaryRange, 0, len(summaries))
	for si := range summaries {
		if s := summaries[si]; s.SectorCount > 0 {
			nonEmpty = append(nonEmpty, s)
		}
	}

	res.Docs = append(res.Docs, &model.MinerSectorSummary{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head, st.Sectors},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
		Detail: model.MinerSectorSummaryDetail{
			Summaries:         nonEmpty,
			CommittedCapacity: minerCommittedCapacity,
		},
	})

	for i := range mds {
		res.Docs = append(res.Docs, &mds[i])
	}

	return nil
}
