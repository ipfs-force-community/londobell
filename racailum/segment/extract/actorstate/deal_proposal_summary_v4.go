package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	market4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/market"
	adt4 "github.com/filecoin-project/specs-actors/v4/actors/util/adt"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("DealProposalSummaryV4", extractDealProposalSummaryV4)
	schema.Register(
		schema.Model{
			Name: "deal-proposal-summary",
			D:    &model.DealProposalSummary{},
		},
	)
}

func extractDealProposalSummaryV4(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *market4.State) error {
	if ticks := ctx.Opts.StateRegular.DealProposalSummaryTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	deals, err := market4.AsDealProposalArray(adt4.WrapStore(ctx.C, ctx.D.ActorStore(ctx.C)), st.Proposals)
	if err != nil {
		return fmt.Errorf("load deal proposal array: %w", err)
	}

	details := []model.DealProposalSummaryDetail{
		model.EmptyDealProposalSummaryDetail(),
		model.EmptyDealProposalSummaryDetail(),
	}

	clients := []map[address.Address]struct{}{
		map[address.Address]struct{}{},
		map[address.Address]struct{}{},
	}

	providers := []map[address.Address]struct{}{
		map[address.Address]struct{}{},
		map[address.Address]struct{}{},
	}

	addProposal := func(p market4.DealProposal) {
		target := 0
		if p.VerifiedDeal {
			target = 1
		}

		clients[target][p.Client] = struct{}{}
		providers[target][p.Provider] = struct{}{}

		details[target].Count++
		details[target].PieceSize = big.Add(details[target].PieceSize, big.NewIntUnsigned(uint64(p.PieceSize)))

		details[target].ProviderCollateral = big.Add(details[target].ProviderCollateral, p.ProviderCollateral)
		details[target].ClientCollateral = big.Add(details[target].ClientCollateral, p.ClientCollateral)
	}

	var out market4.DealProposal
	err = deals.ForEach(&out, func(idx int64) error {
		// ignore expired deals
		if out.EndEpoch < head.Epoch {
			return nil
		}

		addProposal(out)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk through deal proposals: %w", err)
	}

	for di := range details {
		details[di].Clients = uint64(len(clients[di]))
		details[di].Providers = uint64(len(providers[di]))
	}

	id, err := GenRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("ge regular id: %w", err)
	}

	doc := &model.DealProposalSummary{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head, st.Proposals},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
	}
	doc.Detail.Regular = details[0]
	doc.Detail.Verified = details[1]

	res.Docs = append(res.Docs, doc)

	return nil
}
