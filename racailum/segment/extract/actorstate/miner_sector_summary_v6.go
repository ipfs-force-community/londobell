package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	adt6 "github.com/filecoin-project/specs-actors/v6/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	summaryDaysV6 = []abi.ChainEpoch{1, 7, 14, 30}
	for d := (miner6.MinSectorExpiration / builtin6.EpochsInDay); d <= (miner6.MaxSectorExpirationExtension / builtin6.EpochsInDay); d += 180 {
		summaryDaysV6 = append(summaryDaysV6, abi.ChainEpoch(d))
	}

	schema.Register(
		schema.Model{
			Name: "miner-deal-sector",
			D:    &model.MinerDealSector{},
		},
	)
	mustRegisterRegularExtractor("MinerSectorSummaryV6", extractMinerSectorSummaryV6)
}

var summaryDaysV6 []abi.ChainEpoch

func extractMinerSectorSummaryV6(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner6.State) error {
	if ticks := ctx.Opts.StateRegular.MinerSectorSummaryTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	if st.Sectors.Equals(emptyMinerStateV6.Sectors) {
		return nil
	}

	daysMax := summaryDaysV6[len(summaryDaysV6)-1]
	summaries := make([]*model.MinerSectorSummaryRange, 0, len(summaryDaysV6)+1)
	summariesInDays := make([]*model.MinerSectorSummaryRange, 0, int(daysMax+1))

	var prevDays abi.ChainEpoch
	for _, days := range summaryDaysV6 {
		current := &model.MinerSectorSummaryRange{
			LowerBound:              prevDays * builtin6.EpochsInDay,
			UpperBound:              days * builtin6.EpochsInDay,
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
		LowerBound:              prevDays * builtin6.EpochsInDay,
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

	sectors, err := miner6.LoadSectors(adt6.WrapStore(ctx.C, ctx.D.ActorStore(ctx.C)), st.Sectors)
	if err != nil {
		return fmt.Errorf("load sectors from adt store: %w", err)
	}

	var out miner6.SectorOnChainInfo
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
		remainDays := remainDuration / builtin6.EpochsInDay

		idx := int(remainDays)
		target := summariesInDays[idx]
		target.SectorCount++
		target.DealCount += uint64(len(out.DealIDs))
		target.TotalDealWeight = big.Add(target.TotalDealWeight, out.DealWeight)
		target.TotalVerifiedDealWeight = big.Add(target.TotalVerifiedDealWeight, out.VerifiedDealWeight)
		target.TotalInitialPledge = big.Add(target.TotalInitialPledge, out.InitialPledge)

		if out.SealProof < abi.RegisteredSealProof_StackedDrg2KiBV1_1 {
			target.TotalV1InitialPledge = big.Add(target.TotalV1InitialPledge, out.InitialPledge)
			target.V1SectorCount++
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
				QAPower:            miner6.QAPowerForSector(sectorSize, &out),
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
