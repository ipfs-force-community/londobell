package gen

import (
	"context"
	"os"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/testutils"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func Test_extractMinerSectorHealthV6(t *testing.T) {
	ctx := context.Background()
	res := extract.NewRes(0, 0)
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()
	headerCId, _ := cid.Decode("bafy2bzacedjnglismll3py5nx6q6g2lha6xwovrncl6b53mrx5v6m4k4wuc6y")
	actorStore := store.ActorStore(ctx, localBs)
	var out miner6.State
	err = actorStore.Get(ctx, headerCId, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	ectx, _ := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	err = extractMinerSectorHealthV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headerCId},
		Epoch: abi.ChainEpoch(455000),
	}, &out)

	require.NoError(t, err)
}

func Test_GenerateMinerSectorData(t *testing.T) {
	// Generate Data for:
	// 		Network: calibration
	// 		Epoch: 455000
	// 		ActorAddress: t025418
	// 		Head: bafy2bzacedjnglismll3py5nx6q6g2lha6xwovrncl6b53mrx5v6m4k4wuc6y

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzacedjnglismll3py5nx6q6g2lha6xwovrncl6b53mrx5v6m4k4wuc6y")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}
