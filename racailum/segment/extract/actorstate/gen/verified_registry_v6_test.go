package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	verifreg6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/verifreg"
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

func TestExtractVerifRegV6(t *testing.T) {
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

	headCid, _ := cid.Decode(testVerifRegActorCid)
	head := &common.ActorHead{
		Actor: &types.Actor{
			Head: headCid,
		},
		Epoch: abi.ChainEpoch(455000),
	}

	var out verifreg6.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractVerifRegV6(ectx, res, head, &out)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Docs))
	ds := VerifiedRegistryResult()
	pass := false
	for _, doc := range res.Docs {
		vr := doc.(*model.VerifiedRegistry)
		if v, ok := ds[vr.Addr]; ok {
			require.Equal(t, *v, *vr)
			pass = true
		}
	}
	require.Equal(t, true, pass)
}

func VerifiedRegistryResult() map[address.Address]*model.VerifiedRegistry {
	ds := make(map[address.Address]*model.VerifiedRegistry, 1)
	addr1, _ := address.NewFromString("f02284")
	id, _ := cid.Decode("bafy2bzacebafpv56psbv5lfngxrh3ay5xts6wywu3wlfd5kzy54a5woszri6c")
	path1, _ := cid.Decode("bafy2bzacebcevlsbuj3yujjcxv2y23ntmqmaiy24hzcollevlipsurhm5vxte")
	path2, _ := cid.Decode("bafy2bzacednjxkxfyv4zhmxz6qobc3x22dxr3ikikp2np3ztmnwuhrofaqmae")
	epoch := abi.ChainEpoch(455000)

	ds[addr1] = &model.VerifiedRegistry{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{path1, path2},
			Addr:  addr1,
			Epoch: epoch,
		},
		Detail: model.VerifiedRegistryDetail{
			Type: "Verifier",
			Cap:  abi.NewStoragePower(10000000),
		},
	}
	return ds
}
