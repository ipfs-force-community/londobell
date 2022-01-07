package gen

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	power6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/power"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractClaimedPowerV6(t *testing.T) {
	// 1.Mock
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
	headCid, _ := cid.Decode(testPowerActorCid)
	var out power6.State
	err = store.ActorStore(ctx, localBs).Get(ctx, headCid, &out)
	require.NoError(t, err)

	// 2.Call func to test
	err = extractClaimedPowerV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Epoch: abi.ChainEpoch(455000),
	}, &out)

	// 3.Assert result
	require.NoError(t, err)
	require.Equal(t, 118, len(res.Docs))
	ds := ClaimedPowerExpectResult()
	for _, doc := range res.Docs {
		cpw := doc.(*model.ClaimedPower)
		if v, ok := ds[cpw.Addr]; ok {
			require.Equal(t, *v, *cpw)
		}
	}
}
func ClaimedPowerExpectResult() map[address.Address]*model.ClaimedPower {
	ds := make(map[address.Address]*model.ClaimedPower, 1)
	addr1, _ := address.NewFromString("f016066")
	id, _ := cid.Decode("bafy2bzacecb7jykapebydydsvcqyzn4ukctiukqywicfsq3lmq3din5thb7r2")
	path1, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	path2, _ := cid.Decode("bafy2bzaceafax6er2dups5gg2fpb3xjcicux2koydgwpaut3vctpihwfylrqg")
	fmt.Println(id)
	epoch := abi.ChainEpoch(455000)
	wpType := abi.RegisteredPoStProof(8)
	rawPower := abi.NewStoragePower(1133871366144)
	adjPower := abi.NewStoragePower(1133871366144)
	ds[addr1] = &model.ClaimedPower{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr1, Epoch: epoch},
		Detail:            &power6.Claim{WindowPoStProofType: wpType, RawBytePower: rawPower, QualityAdjPower: adjPower},
	}
	return ds
}
