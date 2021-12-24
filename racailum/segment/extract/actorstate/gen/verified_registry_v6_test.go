package gen

import (
	"context"
	"os"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	verifreg6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/verifreg"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/testutils"
)

func Test_extractVerifRegV6(t *testing.T) {
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

	headCid, _ := cid.Decode("bafy2bzacebcevlsbuj3yujjcxv2y23ntmqmaiy24hzcollevlipsurhm5vxte")
	head := &common.ActorHead{
		Actor: &types.Actor{
			Head: headCid,
		},
		Epoch: abi.ChainEpoch(455000),
	}

	var out verifreg6.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractVerifRegV6(ectx, res, head, &out)
	require.NoError(t, err)
	require.NotEmpty(t, res.Docs)
}

func GenerateVerifiedRegistryData(t *testing.T) {
	// Generate Data for:
	//     Network: calibration
	//     Epoch: 455000
	//     ActorAddress: t06
	//     Head: bafy2bzacebcevlsbuj3yujjcxv2y23ntmqmaiy24hzcollevlipsurhm5vxte

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzacebcevlsbuj3yujjcxv2y23ntmqmaiy24hzcollevlipsurhm5vxte")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}
