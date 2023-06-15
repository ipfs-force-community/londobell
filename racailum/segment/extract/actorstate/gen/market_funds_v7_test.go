package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	market7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/market"
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

func TestExtractMarketFundsV7(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	defer func() {
		_ = closer()
	}()
	require.NoError(t, err)
	headCid, err := cid.Decode(testMarketActorCid)
	require.NoError(t, err)
	actorStore := store.ActorStore(ctx, localBs)
	var out market7.State
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	latestDealID := int64(-1)
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions())
	require.NoError(t, err)
	res := extract.NewRes(0, 0)
	addr, _ := address.NewFromString("t05")
	err = extractMarketFundsV7(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Addr:  addr,
		Epoch: abi.ChainEpoch(792000)}, &out)
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
	id, _ := cid.Decode("bafy2bzaceddkuvvuwmm2txammt6gvd7sysjiihbyromaoapqoqfqfzbwpyqds")
	path1, _ := cid.Decode("bafy2bzacebzk4bvrrzhyvzm2gfjymwk7yatpzr56uiwkdgfyzg4evpvgcihiw")
	path2, _ := cid.Decode("bafy2bzacea7aljfqbxndbdwxordz4tcrrpkcfbwpxw27hgoqndi5snxgagkzy")
	addr, _ := address.NewFromString("t05")
	epoch := abi.ChainEpoch(792000)
	ds[addr] = &model.MarketFunds{ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr, Epoch: epoch},
		Detail: model.MarketFundsDetail{
			TotalLocked:                   big.MustFromString("5022438941617358712013"),
			TotalClientLockedCollateral:   big.NewInt(0),
			TotalProviderLockedCollateral: big.NewInt(0),
			TotalClientStorageFee:         big.MustFromString("5022438941617358712013"),

			ClientUnLockCollateralInFuture:   []abi.TokenAmount{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)},
			ProviderUnLockCollateralInFuture: []abi.TokenAmount{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)},
			ClientUnlockStorageFeeInFuture:   []abi.TokenAmount{big.MustFromString("47834250567087012480"), big.MustFromString("287005503402522074880"), big.MustFromString("334839753969609087360"), big.MustFromString("765347971317138293430")},
		},
	}
	return ds
}
