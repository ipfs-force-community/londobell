package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"

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

func TestExtractMinerSectorHealthV6(t *testing.T) {
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
	require.Equal(t, 1, len(res.Docs))
	resultMoc := MinerSectorHealthExpectResult()
	cpw := res.Docs[0].(*model.MinerSectorHealth)
	require.Equal(t, cpw, resultMoc)
}

func MinerSectorHealthExpectResult() *model.MinerSectorHealth {
	var ds = model.MinerSectorHealth{}
	addr1, _ := address.NewFromString("")
	id, _ := cid.Decode("bafy2bzaceatoatxith2aex4t2rlotswi5h6t6z6ayrxaio3rlfeuu6vunnjt2")
	path1, _ := cid.Decode("bafy2bzacedjnglismll3py5nx6q6g2lha6xwovrncl6b53mrx5v6m4k4wuc6y")
	epoch := abi.ChainEpoch(455000)
	ds = model.MinerSectorHealth{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1}, Addr: addr1, Epoch: epoch},
		Detail: model.MinerSectorHealthDetail{
			Faults:                0,
			Recoveries:            0,
			Unproven:              0,
			Active:                8,
			ActiveSectorsQAPower:  abi.NewStoragePower(274877906944),
			FaultsQAPower:         abi.NewStoragePower(0),
			RecoveriesQAPower:     abi.NewStoragePower(0),
			UnprovenQAPower:       abi.NewStoragePower(0),
			ActiveSectorsRawPower: abi.NewStoragePower(274877906944),
			FaultsRawPower:        abi.NewStoragePower(0),
			RecoveriesRawPower:    abi.NewStoragePower(0),
			UnprovenRawPower:      abi.NewStoragePower(0),
			TerminatedSectors:     0,
		},
	}
	return &ds
}
