package gen

import (
	"context"
	"os"

	power6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/power"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"

	//"github.com/ipfs-force-community/londobell/common"
	"go.uber.org/zap"

	"testing"

	"github.com/ipfs-force-community/londobell/testutils"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

func Test_extractClaimedPowerV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	res := extract.NewRes(0, 0)
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)
	claimsCid, _ := cid.Decode("bafy2bzaceafax6er2dups5gg2fpb3xjcicux2koydgwpaut3vctpihwfylrqg")
	headCid, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	err = extractClaimedPowerV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Epoch: abi.ChainEpoch(455000),
	}, &power6.State{Claims: claimsCid})

	require.NoError(t, err)
	require.NotEmpty(t, res.Docs)
}

func GeneratePowerData(t *testing.T) {
	// Generate Data for:
	// 		Network: calibration
	// 		Epoch: 455000
	// 		ActorAddress: t04
	// 		Head: bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay
	// 		Claims: bafy2bzaceafax6er2dups5gg2fpb3xjcicux2koydgwpaut3vctpihwfylrqg

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}
