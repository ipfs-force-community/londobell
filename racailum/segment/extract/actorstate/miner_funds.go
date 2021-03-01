package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	"github.com/ipfs/go-cid"

	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mir"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("MinerFundsV2", extractMinerFundsV2)
	mustRegisterRegularExtractor("MinerFundsV3", extractMinerFundsV3)

	schema.Register(
		schema.Model{
			Name: "miner-funds",
			D:    &model.MinerFunds{},
		},
	)
}

func extractMinerFundsV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner2.State) error {
	return extractMinerFunds(ctx, res, head, st)
}

func extractMinerFundsV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner3.State) error {
	return extractMinerFunds(ctx, res, head, st)
}

func extractMinerFunds(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st interface{}) error {
	if ticks := ctx.Opts.StateRegular.MinerFundsTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	var detail model.MinerFundsDetail
	sum := abi.NewTokenAmount(0)

	switch st := st.(type) {
	case *miner2.State:
		if err := mir.Mirror(&detail, st); err != nil {
			return fmt.Errorf("mirroring *miner2.State: %w", err)
		}

		if !st.VestingFunds.Equals(emptyMinerStateV2.VestingFunds) {
			funds, err := st.LoadVestingFunds(adt2.WrapStore(ctx.C, ctx.D.Store(ctx.C)))
			if err != nil {
				return fmt.Errorf("load vesting funds: %w", err)
			}

			for _, v := range funds.Funds {
				sum = big.Add(sum, v.Amount)
			}
		}

	case *miner3.State:
		if err := mir.Mirror(&detail, st); err != nil {
			return fmt.Errorf("mirroring *miner2.State: %w", err)
		}

		if !st.VestingFunds.Equals(emptyMinerStateV3.VestingFunds) {
			funds, err := st.LoadVestingFunds(adt3.WrapStore(ctx.C, ctx.D.Store(ctx.C)))
			if err != nil {
				return fmt.Errorf("load vesting funds: %w", err)
			}

			for _, v := range funds.Funds {
				sum = big.Add(sum, v.Amount)
			}
		}
	}

	detail.VestingTotal = sum

	// all zero
	if isEmptyOrZero(detail.PreCommitDeposits) &&
		isEmptyOrZero(detail.VestingTotal) &&
		isEmptyOrZero(detail.LockedFunds) &&
		isEmptyOrZero(detail.FeeDebt) &&
		isEmptyOrZero(detail.InitialPledge) {
		return nil
	}

	id, err := genRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("generate regular id: %w", err)
	}

	res.Docs = append(res.Docs, &model.MinerFunds{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
		Detail: detail,
	})

	return nil
}
