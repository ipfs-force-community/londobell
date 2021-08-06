package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
)

func init() {
	mustRegisterRegularExtractor("MinerSectorHealthV5", extractMinerSectorHealthV5)
}

func extractMinerSectorHealthV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner5.State) error {
	if ticks := ctx.Opts.StateRegular.MinerSectorHeathTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	if !st.DeadlineCronActive {
		return nil
	}
	actStore := ctx.D.ActorStore(ctx.C)
	deadlines, err := st.LoadDeadlines(actStore)
	if err != nil {
		return fmt.Errorf("load deadlines failed: %w", err)
	}
	detail := model.MinerSectorHealthDetail{
		ActiveSectorsQAPower: abi.NewTokenAmount(0),
		FaultsQAPower:        abi.NewTokenAmount(0),
		RecoveriesQAPower:    abi.NewTokenAmount(0),
		UnprovenQAPower:      abi.NewTokenAmount(0),
	}
	err = deadlines.ForEach(actStore, func(dlIdx uint64, dl *miner5.Deadline) error {
		if dl == nil {
			return nil
		}

		ps, err := dl.PartitionsArray(actStore)
		if err != nil {
			return fmt.Errorf("get dl partition failed: %w", err)
		}
		var part miner5.Partition
		return ps.ForEach(&part, func(partIdx int64) error {
			detail.ActiveSectorsQAPower = big.Add(detail.ActiveSectorsQAPower, part.ActivePower().QA)
			detail.FaultsQAPower = big.Add(detail.FaultsQAPower, part.FaultyPower.QA)
			detail.RecoveriesQAPower = big.Add(detail.RecoveriesQAPower, part.RecoveringPower.QA)
			detail.UnprovenQAPower = big.Add(detail.UnprovenQAPower, part.UnprovenPower.QA)

			bf, err := part.ActiveSectors()
			if err != nil {
				return fmt.Errorf("load partition active sector failed: %w", err)
			}
			active, err := bf.Count()
			if err != nil {
				return fmt.Errorf("count active bitfield failed: %w", err)
			}
			detail.Active += active

			recoveries, err := part.Recoveries.Count()
			if err != nil {
				return fmt.Errorf("count recoveries bitfield failed: %w", err)
			}
			detail.Recoveries += recoveries

			faults, err := part.Faults.Count()
			if err != nil {
				return fmt.Errorf("count faults bitfield failed: %w", err)
			}
			detail.Faults += faults

			unproven, err := part.Unproven.Count()
			if err != nil {
				return fmt.Errorf("count unproven bitfield failed: %w", err)
			}
			detail.Unproven += unproven

			return nil
		})
	})

	id, err := genRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("ge regular id: %w", err)
	}

	doc := &model.MinerSectorHealth{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
	}
	doc.Detail = detail
	res.Docs = append(res.Docs, doc)
	return nil
}
