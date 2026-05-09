package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	multisig7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/multisig"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractMultisigBalanceV7(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()

	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)

	latestDealID := int64(-1)
	ectx, err := extract.NewCtx(context.Background(), mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions(), nil, nil)
	require.NoError(t, err)

	res := extract.NewRes(0, 0)

	headCid, _ := cid.Decode(testMultisigActorCid)
	addr, _ := address.NewFromString("t011092")
	head := &common.ActorHead{
		Epoch: abi.ChainEpoch(792000),
		Actor: &types.Actor{
			Head:    headCid,
			Balance: big.MustFromString("20000000000000000000"),
		},
		Addr: addr,
	}

	var out multisig7.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractMultisigBalanceV7(ectx, res, head, &out)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Docs))
	ds := MultisigBalanceExpectResult()
	for _, doc := range res.Docs {
		mb := doc.(*model.MultisigBalance)
		if v, ok := ds[mb.Addr]; ok {
			require.Equal(t, *v, *mb)
		}
	}
}

func MultisigBalanceExpectResult() map[address.Address]*model.MultisigBalance {
	ds := make(map[address.Address]*model.MultisigBalance, 1)
	addr1, _ := address.NewFromString("f011092")
	id, _ := cid.Decode("bafy2bzaceckrfp24w77ef535gngwsvpyqbnudkhfenmvfnxkgdhyb6pcneunw")
	path1, _ := cid.Decode("bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a")
	epoch := abi.ChainEpoch(792000)

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
