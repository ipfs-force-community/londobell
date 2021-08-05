package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	market5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/market"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mir"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("MarketFundsV5", extractMarketFundsV5)
	schema.Register(
		schema.Model{
			Name: "market-funds",
			D:    &model.MarketFunds{},
		},
	)
}

func extractMarketFundsV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *market5.State) error {
	if ticks := ctx.Opts.StateRegular.MarketFundsTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	var detail model.MarketFundsDetail
	if err := mir.Mirror(&detail, st); err != nil {
		return fmt.Errorf("mirroring *market.State: %w", err)
	}

	ts := NormalEpochRange(head)
	unlockClientFee := NewTokenAmountArr(len(ts))
	unlockProviderCollateral := NewTokenAmountArr(len(ts))
	unlockClientCollateral := NewTokenAmountArr(len(ts))

	actStore := ctx.D.ActorStore(ctx.C)
	proposals, err := market5.AsDealProposalArray(actStore, st.Proposals)
	deal := &market5.DealProposal{}
	err = proposals.ForEach(deal, func(_ int64) error {
		if deal.StartEpoch >= head.Epoch {
			return nil
		}
		unlockClientFeeTmp := NewTokenAmountArr(len(ts))

		for i := range ts {
			if ts[i] < deal.EndEpoch {
				unlockClientFeeTmp[i] = big.Mul(abi.NewTokenAmount(int64(ts[i]-head.Epoch)), deal.StoragePricePerEpoch)
			} else {
				unlockClientFeeTmp[i] = big.Mul(abi.NewTokenAmount(int64(deal.EndEpoch-head.Epoch)), deal.StoragePricePerEpoch)
			}
		}

		for i := range ts {
			if ts[i] >= deal.EndEpoch {
				unlockClientCollateral[i] = big.Add(unlockClientCollateral[i], deal.ClientCollateral)
				unlockProviderCollateral[i] = big.Add(unlockProviderCollateral[i], deal.ProviderCollateral)
				break
			}
		}

		for i := len(ts) - 1; i > 0; i-- {
			unlockClientFeeTmp[i] = big.Sub(unlockClientFeeTmp[i], unlockClientFeeTmp[i-1])
		}

		for i := range ts {
			unlockClientFee[i] = big.Add(unlockClientFee[i], unlockClientFeeTmp[i])
		}
		return nil
	})

	totalLock := big.Add(big.Add(st.TotalProviderLockedCollateral, st.TotalClientLockedCollateral), st.TotalClientStorageFee)
	detail.TotalLocked = totalLock
	detail.ClientUnLockCollateralInFuture = unlockClientCollateral
	detail.ClientUnlockStorageFeeInFuture = unlockClientFee
	detail.ProviderUnLockCollateralInFuture = unlockProviderCollateral

	id, err := genRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("ge regular id: %w", err)
	}

	doc := &model.MarketFunds{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head, st.Proposals},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
		Detail: detail,
	}
	res.Docs = append(res.Docs, doc)

	return nil
}
