package gen

import (
	"context"
	"os"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/chain/store"
	market6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/market"
	"github.com/ipfs-force-community/londobell/testutils"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func Test_extractDealProposalDetailedV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	defer func() {
		_ = closer()
	}()
	require.NoError(t, err)
	headCid, err := cid.Decode("bafy2bzacectvwb5snunyl2qybrs5hybtnz5l7xug73rf6sowrag7qh6ik2zta")
	require.NoError(t, err)
	actorStore := store.ActorStore(ctx, localBs)
	var out market6.State
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{localBs}
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)
	res := extract.NewRes(0, 0)
	err = extractDealProposalDetailedV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Epoch: abi.ChainEpoch(455000)}, &out)
	require.NoError(t, err)
}

func GenerateMarketData(t *testing.T) {
	// Generate Data for:
	// 		Network: calibration
	// 		Epoch: 455000
	// 		ActorAddress: t05
	// 		Head: bafy2bzacectvwb5snunyl2qybrs5hybtnz5l7xug73rf6sowrag7qh6ik2zta

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzacectvwb5snunyl2qybrs5hybtnz5l7xug73rf6sowrag7qh6ik2zta")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}
