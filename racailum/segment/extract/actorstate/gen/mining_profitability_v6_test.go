package gen

import (
	"context"
	"math/big"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	reward6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/reward"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestMiningProfitabilityV6(t *testing.T) {
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

	pheadCid, _ := cid.Decode(testPowerActorCid)
	pcodeCid, _ := cid.Decode("bafkqaetgnfwc6nrpon2g64tbm5sxa33xmvza")
	powerActor := &types.Actor{
		Code: pcodeCid,
		Head: pheadCid,
	}

	headCid, _ := cid.Decode(testRewardActorCid)
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
	assert.Equal(t, 1, len(res.Docs))
	ds := MiningProfitabilityResult()
	pass := false
	for _, doc := range res.Docs {
		mp := doc.(*model.MiningProfitability)
		if v, ok := ds[mp.Addr]; ok {
			require.Equal(t, *v, *mp)
			pass = true
		}
	}
	require.Equal(t, true, pass)
}

func MiningProfitabilityResult() map[address.Address]*model.MiningProfitability {
	ds := make(map[address.Address]*model.MiningProfitability, 1)
	addr1, _ := address.NewFromString("f02")
	id, _ := cid.Decode("bafy2bzaceae5ndjbrxmcbuuyjoadpipqsqvnaihsty7alm6juwlao3gk5vwvw")
	path1, _ := cid.Decode("bafy2bzacecy72ujmh4ywna4wdm4z5puv6v4664akuibwt7bzz55i74gwriwmo")
	path2, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	epoch := abi.ChainEpoch(455000)
	expectedDayReward := abi.NewTokenAmount(616019864318014729)
	initPledge := abi.NewTokenAmount(999999984306749440)
	filInitialConsensusPledge := new(big.Int)
	filInitialConsensusPledge.SetString("-22905006541780486248", 10)
	initialConsensusPledge := abi.TokenAmount{Int: filInitialConsensusPledge}
	filInitialStoragePledge := new(big.Int)
	filInitialStoragePledge.SetString("23905006526087235688", 10)
	initialStoragePledge := abi.TokenAmount{Int: filInitialStoragePledge}
	filProjectionOfInitialPledge := new(big.Int)
	filProjectionOfInitialPledge.SetString("23905006526087235688", 10)
	projectionOfInitialPledge := abi.TokenAmount{Int: filProjectionOfInitialPledge}
	projectionOfFaultFee := abi.NewTokenAmount(2280401780961463196)
	filMined := new(big.Int)
	filMined.SetString("14232804985935676109996779", 10)
	mined := abi.TokenAmount{Int: filMined}

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
