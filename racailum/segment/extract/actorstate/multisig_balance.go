package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"

	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV2", extractMultisigBalanceV2)
	mustRegisterRegularExtractor("MultisigBalanceV3", extractMultisigBalanceV3)

	schema.Register(
		schema.Model{
			Name: "multisig-balance",
			D:    &model.MultisigBalance{},
		},
	)
}

func extractMultisigBalanceV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig2.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}

func extractMultisigBalanceV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig3.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}

func extractMultisigBalanceDetail(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst interface{}) error {
	epoch := head.Epoch
	var init, locked abi.TokenAmount

	switch st := mst.(type) {
	case *multisig2.State:
		if epoch < st.StartEpoch {
			return nil
		}

		if epoch > st.StartEpoch+st.UnlockDuration {
			return nil
		}

		init = st.InitialBalance
		locked = st.AmountLocked(epoch - st.StartEpoch)

	case *multisig3.State:
		if epoch < st.StartEpoch {
			return nil
		}

		if epoch > st.StartEpoch+st.UnlockDuration {
			return nil
		}

		init = st.InitialBalance
		locked = st.AmountLocked(epoch - st.StartEpoch)
	}

	vested := big.Sub(init, locked)

	id, err := genRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}

	res.Docs = append(res.Docs, &model.MultisigBalance{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
		Detail: model.MultisigBalanceDetail{
			Locked: locked,
			Vested: vested,
		},
	})

	return nil
}
