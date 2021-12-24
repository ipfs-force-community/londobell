package gen

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	reward6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/reward"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/testutils"
)

func Test_miningProfitabilityV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()

	mockDAL := &MockDAL{}
	mockDAL.On("ChainBlockstore").Return(localBs)

	ectx, err := extract.NewCtx(context.Background(), mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)

	res := extract.NewRes(0, 0)

	pheadCid, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	pcodeCid, _ := cid.Decode("bafkqaetgnfwc6nrpon2g64tbm5sxa33xmvza")
	powerActor := &types.Actor{
		Code: pcodeCid,
		Head: pheadCid,
	}

	headCid, _ := cid.Decode("bafy2bzacecy72ujmh4ywna4wdm4z5puv6v4664akuibwt7bzz55i74gwriwmo")
	addr, _ := address.NewFromString("t02")
	fil := new(big.Int)
	filCirculating, _ := fil.SetString("-9361629299923245362566404", 10)
	head := &common.ActorHead{
		Actor: &types.Actor{
			Head: headCid,
		},
		Global: struct{ Power *types.Actor }{Power: powerActor},
		Addr:   addr,
		Epoch:  abi.ChainEpoch(455000),
		CirculatingSupply: &api.CirculatingSupply{
			FilCirculating: abi.TokenAmount{Int: filCirculating},
		},
	}

	var out reward6.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractMiningProfitabilityV6(ectx, res, head, &out)
	require.NoError(t, err)
	require.NotEmpty(t, res.Docs)
}

func GenerateRewardData(t *testing.T) {
	// Generate Data for:
	//     Network: calibration
	//     Epoch: 455000
	//     ActorAddress: t02
	//     Head: bafy2bzacecy72ujmh4ywna4wdm4z5puv6v4664akuibwt7bzz55i74gwriwmo

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzacecy72ujmh4ywna4wdm4z5puv6v4664akuibwt7bzz55i74gwriwmo")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}

//func GeneratePowerData(t *testing.T) {
//	// Generate Data for:
//	//     Network: calibration
//	//     Epoch: 455000
//	//     ActorAddress: t04
//	//     Head: bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay //??
//
//	//url := os.Getenv("TEST_LOTUS_URL")
//	ctx := context.Background()
//	rootCid, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay") //actorhead:actorstate??
//	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
//	require.NoError(t, err)
//	localBS, _, err := testutils.NewLocalBlockStore(ctx)
//	require.NoError(t, err)
//	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
//	require.NoError(t, err)
//}
