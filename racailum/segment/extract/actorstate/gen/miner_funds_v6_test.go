package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
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

func TestExtractMinerFundsV6(t *testing.T) {
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
	err = extractMinerFundsV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headerCId},
		Epoch: abi.ChainEpoch(455000),
	}, &out)

	require.NoError(t, err)
	require.Equal(t, 1, len(res.Docs))
	resultMoc := MinerFundsExpectResult()
	cpw := res.Docs[0].(*model.MinerFunds)
	require.Equal(t, cpw, resultMoc)
}

func MinerFundsExpectResult() *model.MinerFunds {
	var ds = model.MinerFunds{}
	addr1, _ := address.NewFromString("")
	id, _ := cid.Decode("bafy2bzaceatoatxith2aex4t2rlotswi5h6t6z6ayrxaio3rlfeuu6vunnjt2")
	path1, _ := cid.Decode("bafy2bzacedjnglismll3py5nx6q6g2lha6xwovrncl6b53mrx5v6m4k4wuc6y")
	epoch := abi.ChainEpoch(455000)
	preCommitDeposits := big.MustFromString("137662762021001930325")
	lockedFunds := big.MustFromString("5170643750613832130")
	feeDebt := abi.NewTokenAmount(0)
	initialPledge := big.MustFromString("7999999874453995520")
	vestInFuture := []abi.TokenAmount{
		big.MustFromString("28896591741636207"),
		big.MustFromString("173379550449817244"),
		big.MustFromString("202276142191453451"),
		big.MustFromString("462345467866179317"),
	}
	pledgeRelease := []abi.TokenAmount{
		abi.NewTokenAmount(0),
		abi.NewTokenAmount(0),
		abi.NewTokenAmount(0),
		abi.NewTokenAmount(0),
	}
	ds = model.MinerFunds{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1}, Addr: addr1, Epoch: epoch},
		Detail: model.MinerFundsDetail{
			PreCommitDeposits: preCommitDeposits,
			LockedFunds:       lockedFunds,
			FeeDebt:           feeDebt,
			InitialPledge:     initialPledge,
			VestInFuture:      vestInFuture,
			PledgeRelease:     pledgeRelease,
		},
	}
	return &ds
}
