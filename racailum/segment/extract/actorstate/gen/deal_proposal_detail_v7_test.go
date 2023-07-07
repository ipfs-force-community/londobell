package gen

import (
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

func TestExtractDealProposalDetailedV7(t *testing.T) {
	//ctx := context.Background()
	//localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	//defer func() {
	//	_ = closer()
	//}()
	//require.NoError(t, err)
	//headCid, err := cid.Decode(testMarketActorCid)
	//require.NoError(t, err)
	//addr, _ := address.NewFromString("f05")
	//actorStore := store.ActorStore(ctx, localBs)
	//var out market7.State
	//err = actorStore.Get(ctx, headCid, &out)
	//require.NoError(t, err)
	//mockDAL := &MockDAL{}
	//mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	//latestDealID := int64(-1)
	//ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions())
	//require.NoError(t, err)
	//res := extract.NewRes(0, 0)
	//
	//c, err := cid.Decode("bafy2bzacedqzqskx4hjrwza2ey75mhctmp66cmjpn3q5udcyimyffls76vokm")
	//require.NoError(t, err)
	//
	//err = extractDealProposalDetailedV7(ectx, res, &common.ActorHead{
	//	Actor: &types.Actor{Head: headCid},
	//	Addr:  addr,
	//	Epoch: abi.ChainEpoch(792000), Tsk: types.NewTipSetKey(c)}, &out)
	//
	//require.NoError(t, err)
	//require.Equal(t, 15790, len(res.Docs))
	//ds := DealProposalDetailExpectResult()
	//for _, doc := range res.Docs {
	//	dp, ok := doc.(*model.DealProposalDetail)
	//	if ok {
	//		if v, ok := ds[dp.Addr]; ok {
	//			require.Equal(t, *v, *dp)
	//		}
	//	}
	//}
}

func DealProposalDetailExpectResult() map[address.Address]*model.DealProposalDetail {
	ds := make(map[address.Address]*model.DealProposalDetail, 1)
	id, _ := cid.Decode("bafy2bzaceddkuvvuwmm2txammt6gvd7sysjiihbyromaoapqoqfqfzbwpyqds")
	path1, _ := cid.Decode("bafy2bzacebzk4bvrrzhyvzm2gfjymwk7yatpzr56uiwkdgfyzg4evpvgcihiw")
	path2, _ := cid.Decode("bafy2bzacea7aljfqbxndbdwxordz4tcrrpkcfbwpxw27hgoqndi5snxgagkzy")
	addr, _ := address.NewFromString("f030477")
	epoch := abi.ChainEpoch(792000)
	ds[addr] = &model.DealProposalDetail{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1, path2}, Addr: addr, Epoch: epoch},
		Detail: model.MinerDealProposalDetail{
			UnVerifiedDealCount:    uint64(3),
			UnVerifiedDealEndCount: uint64(0),
			VerifiedDealCount:      uint64(0),
			VerifiedDealEndCount:   uint64(0),
		},
	}
	return ds
}
