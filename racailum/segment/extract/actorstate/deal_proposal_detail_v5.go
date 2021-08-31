package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	market5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/market"
	adt5 "github.com/filecoin-project/specs-actors/v5/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("DealProposalDetailedV5", extractDealProposalDetailedV5)
	schema.Register(
		schema.Model{
			Name: "deal-proposal-detail",
			D:    &model.DealProposalDetail{},
		},
	)
}

func extractDealProposalDetailedV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *market5.State) error {
	if ticks := ctx.Opts.StateRegular.DealProposalDetailTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	deals, err := market5.AsDealProposalArray(adt5.WrapStore(ctx.C, ctx.D.ActorStore(ctx.C)), st.Proposals)
	if err != nil {
		return fmt.Errorf("load deal proposal array: %w", err)
	}

	dealsStateArr, err := market5.AsDealStateArray(adt5.WrapStore(ctx.C, ctx.D.ActorStore(ctx.C)), st.States)
	if err != nil {
		return fmt.Errorf("load deal state array: %w", err)
	}
	id, err := genRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("ge regular id: %w", err)
	}

	details := map[address.Address]*model.DealProposalDetail{}

	var out market5.DealProposal
	err = deals.ForEach(&out, func(idx int64) error {
		if _, ok := details[out.Provider]; !ok {
			details[out.Provider] = &model.DealProposalDetail{
				ActorStateExBasic: model.ActorStateExBasic{
					ID:    id,
					Path:  []cid.Cid{head.Head, st.Proposals},
					Addr:  out.Provider,
					Epoch: head.Epoch,
				},
			}
		}

		dealState, found, err := dealsStateArr.Get(abi.DealID(idx))
		if err != nil {
			return fmt.Errorf("load deal state failed: %w", err)
		}

		// no matter expire or slash
		unExpired := out.EndEpoch < head.Epoch || (found && dealState.SlashEpoch != -1)
		if unExpired && out.VerifiedDeal {
			details[out.Provider].Detail.VerifiedDealCount++
		} else if unExpired {
			details[out.Provider].Detail.UnVerifiedDealCount++
		} else if out.VerifiedDeal {
			details[out.Provider].Detail.VerifiedDealEndCount++
		} else {
			details[out.Provider].Detail.UnVerifiedDealEndCount++
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk through deal proposals: %w", err)
	}

	for i := range details {
		res.Docs = append(res.Docs, details[i])
	}

	return nil
}
