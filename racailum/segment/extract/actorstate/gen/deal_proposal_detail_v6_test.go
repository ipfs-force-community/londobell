package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	market6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/market"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractDealProposalDetailedV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	defer func() {
		_ = closer()
	}()
	require.NoError(t, err)
	headCid, err := cid.Decode(testMarketActorCid)
	require.NoError(t, err)
	actorStore := store.ActorStore(ctx, localBs)
	var out market6.State
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)
	res := extract.NewRes(0, 0)

	err = extractDealProposalDetailedV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Epoch: abi.ChainEpoch(455000)}, &out)

	require.NoError(t, err)
	require.Equal(t, 27414, len(res.Docs))
	ds := DealProposalDetailExpectResult()
	for _, doc := range res.Docs {
		dp, ok := doc.(*model.DealProposalDetail)
		if ok {
			if v, ok := ds[dp.Addr]; ok {
				require.Equal(t, *v, *dp)
			}
		}
	}
}

func DealProposalDetailExpectResult() map[address.Address]*model.DealProposalDetail {
	ds := make(map[address.Address]*model.DealProposalDetail, 1)
	id, _ := cid.Decode("bafy2bzacediinythyv25eyfe7bdmilptigr6nymklpqili42nfzqqhperpjrk")
	path1, _ := cid.Decode("bafy2bzacectvwb5snunyl2qybrs5hybtnz5l7xug73rf6sowrag7qh6ik2zta")
	path2, _ := cid.Decode("bafy2bzacecwfif5gwqwp7b4ftftlhphatvty6uzykcyh2jemc5fifhuj26e4i")
	addr, _ := address.NewFromString("f024557")
	epoch := abi.ChainEpoch(455000)
	ds[addr] = &model.DealProposalDetail{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr, Epoch: epoch},
		Detail: model.MinerDealProposalDetail{
			UnVerifiedDealCount:    uint64(56),
			UnVerifiedDealEndCount: uint64(0),
			VerifiedDealCount:      uint64(0),
			VerifiedDealEndCount:   uint64(0),
		},
	}
	return ds
}
