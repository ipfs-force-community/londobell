package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"

	"github.com/filecoin-project/lotus/chain/store"
	market6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/market"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractMarketFundsV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	defer func() {
		_ = closer()
	}()
	require.NoError(t, err)
	headCid, err := cid.Decode(testMarketActorCid)
	require.NoError(t, err)
	actorStore := store.ActorStore(ctx, localBs)
	var out market6.State
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)
	res := extract.NewRes(0, 0)
	addr, _ := address.NewFromString("t05")
	err = extractMarketFundsV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Addr:  addr,
		Epoch: abi.ChainEpoch(455000)}, &out)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Docs))

	ds := MarketFundsExpectResult()
	for _, doc := range res.Docs {
		mf := doc.(*model.MarketFunds)
		if v, ok := ds[mf.Addr]; ok {
			require.Equal(t, *v, *mf)
		}
	}

}

func MarketFundsExpectResult() map[address.Address]*model.MarketFunds {
	ds := make(map[address.Address]*model.MarketFunds, 1)
	id, _ := cid.Decode("bafy2bzacedwrrpnxpvj4kw5b72q2kq7kl6ejfsxm2kuytmiwc4ds3hb4xs2rs")
	path1, _ := cid.Decode("bafy2bzacectvwb5snunyl2qybrs5hybtnz5l7xug73rf6sowrag7qh6ik2zta")
	path2, _ := cid.Decode("bafy2bzacecwfif5gwqwp7b4ftftlhphatvty6uzykcyh2jemc5fifhuj26e4i")
	addr, _ := address.NewFromString("t05")
	epoch := abi.ChainEpoch(455000)
	ds[addr] = &model.MarketFunds{ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr, Epoch: epoch},
		Detail: model.MarketFundsDetail{
			TotalLocked:                   big.MustFromString("89584023836529126618"),
			TotalClientLockedCollateral:   big.NewInt(0),
			TotalProviderLockedCollateral: big.NewInt(0),
			TotalClientStorageFee:         big.MustFromString("89584023836529126618"),

			ClientUnLockCollateralInFuture:   []abi.TokenAmount{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)},
			ProviderUnLockCollateralInFuture: []abi.TokenAmount{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)},
			ClientUnlockStorageFeeInFuture:   []abi.TokenAmount{big.MustFromString("420196563484583040"), big.MustFromString("2521179380907498240"), big.MustFromString("2941375944392081280"), big.MustFromString("6723145015753328640")},
		},
	}
	return ds
}
