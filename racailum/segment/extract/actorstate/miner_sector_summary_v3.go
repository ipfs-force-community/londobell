package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func init() {
	summaryDaysV3 = []abi.ChainEpoch{1, 2, 3, 7, 14, 30, 60, 120}
	for d := (miner3.MinSectorExpiration / builtin3.EpochsInDay); d <= (miner3.MaxSectorExpirationExtension / builtin3.EpochsInDay); d += 180 {
		summaryDaysV3 = append(summaryDaysV3, abi.ChainEpoch(d))
	}

	mustRegisterRegularExtractor("MinerSectorSummaryV3", extractMinerSectorSummaryV3)

	schema.Register(
		schema.Model{
			Name: "miner-sector-summary",
			D:    &model.MinerSectorSummary{},
		},
	)
}

var summaryDaysV3 []abi.ChainEpoch

func extractMinerSectorSummaryV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner3.State) error {
	if ticks := ctx.Opts.StateRegular.MinerSectorSummaryTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	if st.Sectors.Equals(emptyMinerStateV3.Sectors) {
		return nil
	}

	daysMax := summaryDaysV3[len(summaryDaysV3)-1]
	summaries := make([]*model.MinerSectorSummaryRange, 0, len(summaryDaysV3)+1)
	summariesInDays := make([]*model.MinerSectorSummaryRange, 0, int(daysMax+1))

	var prevDays abi.ChainEpoch
	for _, days := range summaryDaysV3 {
		current := &model.MinerSectorSummaryRange{
			LowerBound:              prevDays * builtin3.EpochsInDay,
			UpperBound:              days * builtin3.EpochsInDay,
			SectorCount:             0,
			DealCount:               0,
			TotalDealWeight:         big.NewInt(0),
			TotalVerifiedDealWeight: big.NewInt(0),
			TotalInitialPledge:      abi.NewTokenAmount(0),
		}

		summaries = append(summaries, current)
		for i := abi.ChainEpoch(0); i < (days - prevDays); i++ {
			summariesInDays = append(summariesInDays, current)
		}

		prevDays = days
	}

	last := &model.MinerSectorSummaryRange{
		LowerBound:              prevDays * builtin3.EpochsInDay,
		UpperBound:              -1,
		SectorCount:             0,
		DealCount:               0,
		TotalDealWeight:         big.NewInt(0),
		TotalVerifiedDealWeight: big.NewInt(0),
		TotalInitialPledge:      abi.NewTokenAmount(0),
	}

	summaries = append(summaries, last)
	summariesInDays = append(summariesInDays, last)

	sectors, err := miner3.LoadSectors(adt3.WrapStore(ctx.C, ctx.D.ActorStore(ctx.C)), st.Sectors)
	if err != nil {
		return fmt.Errorf("load sectors from adt store: %w", err)
	}

	var out miner3.SectorOnChainInfo
	err = sectors.ForEach(&out, func(n int64) error {
		if out.Expiration <= head.Epoch {
			return nil
		}

		remainDuration := out.Expiration - head.Epoch
		remainDays := remainDuration / builtin3.EpochsInDay

		idx := int(remainDays)
		target := summariesInDays[idx]
		target.SectorCount++
		target.DealCount += uint64(len(out.DealIDs))
		target.TotalDealWeight = big.Add(target.TotalDealWeight, out.DealWeight)
		target.TotalVerifiedDealWeight = big.Add(target.TotalVerifiedDealWeight, out.VerifiedDealWeight)
		target.TotalInitialPledge = big.Add(target.TotalInitialPledge, out.InitialPledge)

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
			Summaries: nonEmpty,
		},
	})

	return nil
}
