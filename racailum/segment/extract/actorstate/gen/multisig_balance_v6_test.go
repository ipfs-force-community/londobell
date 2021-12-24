package gen

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	multisig6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/multisig"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/testutils"
)

func Test_extractMultisigBalanceV6(t *testing.T) {
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

	headCid, _ := cid.Decode("bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a")
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

	fmt.Printf("out:%+v", out)

	err = extractMultisigBalanceV6(ectx, res, head, &out)
	require.NoError(t, err)
	require.NotEmpty(t, res.Docs)
}

func GenerateMultisigData(t *testing.T) {
	// Generate Data for:
	//     Network: calibration
	//     Epoch: 455000
	//     ActorAddress: t011092
	//     Head: bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}
