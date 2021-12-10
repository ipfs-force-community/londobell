package gen

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	market6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/market"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/testutils"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
)

func Test_extractMarketFundsV6(t *testing.T) {
	ctx := context.Background()
	res := extract.NewRes(0, 0)
	localBs, closer,err:= testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()
	headerCId, _ := cid.Decode("bafy2bzacectvwb5snunyl2qybrs5hybtnz5l7xug73rf6sowrag7qh6ik2zta")
	actorStore := store.ActorStore(ctx,localBs)
	var out market6.State
	err = actorStore.Get(ctx,headerCId,&out)
	require.NoError(t, err)
	deals,err := market6.AsDealProposalArray(actorStore,out.Proposals)
	deals.ForEach(nil, func(i int64) error {
		t.Log(i)
		return nil
	})
	mockDAL := &MockDAL{localBs}
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	err = extractMarketFundsV6(ectx,res,&common.ActorHead{
		Actor: &types.Actor{Head: headerCId},
		Epoch: abi.ChainEpoch(455000),
	},&out)

	require.NoError(t, err)
}

