package gen

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	power7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/power"
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

func TestExtractClaimedPowerV7(t *testing.T) {
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
	latestDealID := int64(0)
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions())
	require.NoError(t, err)
	headCid, _ := cid.Decode(testPowerActorCid)
	var out power7.State
	err = store.ActorStore(ctx, localBs).Get(ctx, headCid, &out)
	require.NoError(t, err)

	// 2.Call func to test
	err = extractClaimedPowerV7(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Epoch: abi.ChainEpoch(792000),
	}, &out)

	// 3.Assert result
	require.NoError(t, err)
	require.Equal(t, 94, len(res.Docs))
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
	addr1, _ := address.NewFromString("f033613")
	id, _ := cid.Decode("bafy2bzaceax67h3b54cn6wyppyajyu5tjlr5fl33rm4ngshha6vd2vcykh5fi")
	path1, _ := cid.Decode("bafy2bzaceavqgbrf6tasxwsh33efuy5wepadh3bodjacseqdex2qpn7tlh5fk")
	path2, _ := cid.Decode("bafy2bzaceb5aylgnltzithyiligxoxhf4vxmm36fkzytmynsbin2uabkwmley")
	fmt.Println(id)
	epoch := abi.ChainEpoch(792000)
	wpType := abi.RegisteredPoStProof(8)
	rawPower := abi.NewStoragePower(3298534883328)
	adjPower := abi.NewStoragePower(3298534883328)
	ds[addr1] = &model.ClaimedPower{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr1, Epoch: epoch},
		Detail:            &power7.Claim{WindowPoStProofType: wpType, RawBytePower: rawPower, QualityAdjPower: adjPower},
	}
	return ds
}
