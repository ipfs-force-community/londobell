package gen

import (
	"context"
	"math/big"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	multisig6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/multisig"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractMultisigBalanceV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()

	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)

	ectx, err := extract.NewCtx(context.Background(), mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)

	res := extract.NewRes(0, 0)

	headCid, _ := cid.Decode(testMultisigActorCid)
	addr, _ := address.NewFromString("t011092")
	fil := new(big.Int)
	balance, _ := fil.SetString("+20000000000000000000", 10)
	head := &common.ActorHead{
		Epoch: abi.ChainEpoch(455000),
		Actor: &types.Actor{
			Head:    headCid,
			Balance: types.BigInt{Int: balance},
		},
		Addr: addr,
	}

	var out multisig6.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractMultisigBalanceV6(ectx, res, head, &out)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Docs))
	ds := MultisigBalanceResult()
	pass := false
	for _, doc := range res.Docs {
		mb := doc.(*model.MultisigBalance)
		if v, ok := ds[mb.Addr]; ok {
			require.Equal(t, *v, *mb)
			pass = true
		}
	}
	require.Equal(t, true, pass)
}

func MultisigBalanceResult() map[address.Address]*model.MultisigBalance {
	ds := make(map[address.Address]*model.MultisigBalance, 1)
	addr1, _ := address.NewFromString("f011092")
	id, _ := cid.Decode("bafy2bzacebfmqgdkemrpzoyjsansjaokjreso7tupopywqmxc6z55mralul2w")
	path1, _ := cid.Decode("bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a")
	epoch := abi.ChainEpoch(455000)

	ds[addr1] = &model.MultisigBalance{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{path1},
			Addr:  addr1,
			Epoch: epoch,
		},
		Detail: model.MultisigBalanceDetail{
			Locked:       abi.NewTokenAmount(0),
			Vested:       abi.NewTokenAmount(0),
			VestInFuture: []abi.TokenAmount{abi.NewTokenAmount(0), abi.NewTokenAmount(0), abi.NewTokenAmount(0), abi.NewTokenAmount(0)},
		},
	}
	return ds
}
