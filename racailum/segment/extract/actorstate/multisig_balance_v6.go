package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	multisig6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/multisig"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV6", extractMultisigBalanceV6)

}

func extractMultisigBalanceV6(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *multisig6.State) error {
	epoch := head.Epoch
	var init, locked abi.TokenAmount
	dayList := NormalEpochRange(head)
	vestInFuture := make([]abi.TokenAmount, len(dayList), len(dayList))

	init = st.InitialBalance
	locked = st.AmountLocked(epoch - st.StartEpoch)
	locked = big.Max(locked, big.Zero())

	for i := range dayList {
		vestInFuture[i] = big.Sub(locked, st.AmountLocked(dayList[i]-st.StartEpoch))
		vestInFuture[i] = big.Max(vestInFuture[i], big.Zero())
		vestInFuture[i] = big.Min(vestInFuture[i], head.Balance)
	}

	for i := len(vestInFuture) - 1; i > 0; i-- {
		vestInFuture[i] = big.Sub(vestInFuture[i], vestInFuture[i-1])
	}

	vested := big.Sub(init, locked)
	vested = big.Max(vested, big.Zero())
	locked = big.Min(locked, head.Balance)

	id, err := GenRegularHeadID(head.Head, head.Addr, head.Epoch)
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
			Locked:       locked,
			Vested:       vested,
			VestInFuture: vestInFuture,
		},
	})

	return nil
}
