package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	miner7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractMinerSectorHealthV7(t *testing.T) {
	ctx := context.Background()
	res := extract.NewRes(0, 0)
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()
	headerCId, _ := cid.Decode(testMinerSectorActorCid)
	addr, _ := address.NewFromString("f031582")
	actorStore := store.ActorStore(ctx, localBs)
	var out miner7.State
	err = actorStore.Get(ctx, headerCId, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	latestDealID := int64(-1)
	ectx, _ := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions())
	err = extractMinerSectorHealthV7(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headerCId},
		Addr:  addr,
		Epoch: abi.ChainEpoch(792000),
	}, &out)

	require.NoError(t, err)
	require.Equal(t, 1, len(res.Docs))
	resultMoc := MinerSectorHealthExpectResult()
	cpw := res.Docs[0].(*model.MinerSectorHealth)
	require.Equal(t, cpw, resultMoc)
}

func MinerSectorHealthExpectResult() *model.MinerSectorHealth {
	var ds = model.MinerSectorHealth{}
	addr1, _ := address.NewFromString("f031582")
	id, _ := cid.Decode("bafy2bzacea3kn2b3f5xedjfqsv4xdnveek47glbsqnzsuomzoiqo5lndmaitm")
	path1, _ := cid.Decode("bafy2bzacebtzupwrfw2fgmfqzafht4imbhbks2fkaqfs77ay2cc63au3jap3c")
	epoch := abi.ChainEpoch(792000)
	ds = model.MinerSectorHealth{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1}, Addr: addr1, Epoch: epoch},
		Detail: model.MinerSectorHealthDetail{
			Faults:                0,
			Recoveries:            0,
			Unproven:              0,
			Active:                198,
			Live:                  198,
			All:                   198,
			ActiveSectorsQAPower:  abi.NewStoragePower(6803228196864),
			FaultsQAPower:         abi.NewStoragePower(0),
			RecoveriesQAPower:     abi.NewStoragePower(0),
			UnprovenQAPower:       abi.NewStoragePower(0),
			ActiveSectorsRawPower: abi.NewStoragePower(6803228196864),
			FaultsRawPower:        abi.NewStoragePower(0),
			RecoveriesRawPower:    abi.NewStoragePower(0),
			UnprovenRawPower:      abi.NewStoragePower(0),
			TerminatedSectors:     0,
		},
	}
	return &ds
}
