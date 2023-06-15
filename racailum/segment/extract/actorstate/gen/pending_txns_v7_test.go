package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	multisig7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/multisig"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lotus/chain/store"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractPendingTxnsV7(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()

	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)

	latestDealID := int64(0)
	ectx, err := extract.NewCtx(context.Background(), mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions())
	require.NoError(t, err)

	res := extract.NewRes(0, 0)

	headCid, _ := cid.Decode(testPendingTxnsActorCid)
	head := &common.ActorHead{
		Actor: &types.Actor{
			Head: headCid,
		},
		Epoch: abi.ChainEpoch(792000),
	}

	var out multisig7.State
	actorStore := store.ActorStore(ctx, localBs)
	err = actorStore.Get(ctx, headCid, &out)
	require.NoError(t, err)

	err = extractPendingTxnsV7(ectx, res, head, &out)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res.Docs))
	ds := PendingTxnsExpectResult()
	for _, doc := range res.Docs {
		vr := doc.(*model.PendingTxns)
		if v, ok := ds[vr.Addr]; ok {
			require.Equal(t, *v, *vr)
		}
	}
}

func PendingTxnsExpectResult() map[address.Address]*model.PendingTxns {
	ds := make(map[address.Address]*model.PendingTxns, 1)
	addr1, _ := address.NewFromString("f027502")
	id, _ := cid.Decode("bafy2bzacecy4r7vnw745sczvd4emqozfaijw6j45nmihj6vx4bs5xdtvjdubq")
	path1, _ := cid.Decode("bafy2bzacedmi47gmsb4wyhlpctizrv4bedex2frpi47leufximpkx37kvmrvk")
	path2, _ := cid.Decode("bafy2bzacecmvhaw2qo6yqykbhpb4kbcv3dqyirn7emvsiwg4sjmsn6bbiudku")
	epoch := abi.ChainEpoch(792000)

	to, _ := address.NewFromString("1tl4uuypuo4tyvdwyqisjatvyywkhwntvmvsrkty")
	approved1, _ := address.NewFromString("015229")

	ds[addr1] = &model.PendingTxns{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{path1, path2},
			Addr:  addr1,
			Epoch: epoch,
		},
		Detail: model.PendingTxnsDetail{
			TxnID:    1,
			To:       to,
			Value:    big.MustFromString("200000000000000000"),
			Approved: []address.Address{approved1},
		},
	}
	return ds
}
