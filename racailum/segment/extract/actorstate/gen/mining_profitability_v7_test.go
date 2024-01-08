package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	reward7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/reward"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-state-types/big"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestMiningProfitabilityV7(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()

	mockDAL := &MockDAL{}
	mockDAL.On("ChainBlockstore").Return(localBs)

	latestDealID := int64(-1)
	ectx, err := extract.NewCtx(context.Background(), mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions(), nil)
	require.NoError(t, err)

	res := extract.NewRes(0, 0)

	pheadCid, _ := cid.Decode(testPowerActorCid)
	pcodeCid, _ := cid.Decode("bafkqaetgnfwc6nzpon2g64tbm5sxa33xmvza")
	powerActor := &types.Actor{
		Code: pcodeCid,
		Head: pheadCid,
	}

	headCid, _ := cid.Decode(testRewardActorCid)
	addr, _ := address.NewFromString("t02")
	head := &common.ActorHead{
		Actor: &types.Actor{
			Head: headCid,
		},
		Global: struct{ Power *types.Actor }{Power: powerActor},
		Addr:   addr,
		Epoch:  abi.ChainEpoch(792000),
		CirculatingSupply: &api.CirculatingSupply{
			FilCirculating: abi.NewTokenAmount(0),
		},
	}

	var out reward7.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractMiningProfitabilityV7(ectx, res, head, &out)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Docs))
	ds := MiningProfitabilityExpectResult()
	for _, doc := range res.Docs {
		mp := doc.(*model.MiningProfitability)
		if v, ok := ds[mp.Addr]; ok {
			require.Equal(t, *v, *mp)
		}
	}
}

func MiningProfitabilityExpectResult() map[address.Address]*model.MiningProfitability {
	ds := make(map[address.Address]*model.MiningProfitability, 1)
	addr1, _ := address.NewFromString("f02")
	id, _ := cid.Decode("bafy2bzacebxxbma2ssuiyxkzy4czl6rtonhbzy4yugk7febzzgccgwe5cpzgw")
	path1, _ := cid.Decode("bafy2bzacebwol6ndjdyw7bwfvtctydbc32cik6xkm4wousgg7erejhogakpjq")
	path2, _ := cid.Decode("bafy2bzaceavqgbrf6tasxwsh33efuy5wepadh3bodjacseqdex2qpn7tlh5fk")
	epoch := abi.ChainEpoch(792000)
	expectedDayReward := big.MustFromString("1082899198884490003")
	initPledge := big.MustFromString("999999984306749440")
	initialConsensusPledge := big.MustFromString("-18128292196465084955")
	initialStoragePledge := big.MustFromString("19128292180771834395")
	projectionOfInitialPledge := big.MustFromString("19128292180771834395")
	projectionOfFaultFee := big.MustFromString("3733314356567894993")
	mined := big.MustFromString("24761639842677885214689262")

	ds[addr1] = &model.MiningProfitability{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr1, Epoch: epoch},
		Detail: model.MiningProfitabilityDetail{
			ExpectedDayReward:         expectedDayReward,
			InitialPledge:             initPledge,
			InitialConsensusPledge:    initialConsensusPledge,
			InitialStoragePledge:      initialStoragePledge,
			ProjectionOfInitialPledge: projectionOfInitialPledge,
			ProjectionOfFaultFee:      projectionOfFaultFee,
			Mined:                     mined,
		},
	}
	return ds
}
