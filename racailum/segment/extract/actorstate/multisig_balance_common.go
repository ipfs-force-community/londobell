package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	multisig5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/multisig"
	"github.com/ipfs/go-cid"

	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"

	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"

	multisig4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/multisig"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	schema.Register(
		schema.Model{
			Name: "multisig-balance",
			D:    &model.MultisigBalance{},
		},
	)
}

func extractMultisigBalanceDetail(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst interface{}) error {
	epoch := head.Epoch
	var init, locked abi.TokenAmount
	dayList := []int{1, 7, 14, 30}
	vestInFuture := make([]abi.TokenAmount, len(dayList), len(dayList))

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

	case *multisig4.State:
		if epoch < st.StartEpoch {
			return nil
		}

		if epoch > st.StartEpoch+st.UnlockDuration {
			return nil
		}

		init = st.InitialBalance
		locked = st.AmountLocked(epoch - st.StartEpoch)
	case *multisig5.State:
		init = st.InitialBalance
		locked = st.AmountLocked(epoch - st.StartEpoch)

		for i := range dayList {
			vestInFuture[i] = big.Sub(locked, st.AmountLocked(epoch+abi.ChainEpoch(dayList[i])*builtin.EpochsInDay-st.StartEpoch))
		}

		for i := len(vestInFuture) - 1; i > 0; i-- {
			vestInFuture[i] = big.Sub(vestInFuture[i], vestInFuture[i-1])
		}
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
			Locked:       locked,
			Vested:       vested,
			VestInFuture: vestInFuture,
		},
	})

	return nil
}
