package gen

import (
	"context"
	"os"

	power6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/power"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"

	//"github.com/ipfs-force-community/londobell/common"
	"go.uber.org/zap"

	"testing"

	"github.com/ipfs-force-community/londobell/testutils"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/blockstore"
	bstore "github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type MockDAL struct {
	blockStore blockstore.Blockstore
}

func (dal *MockDAL) ExecutionTrace(ctx context.Context, ts *types.TipSet) (cid.Cid, []*api.InvocResult, error) {
	panic("not impl")
}
func (dal *MockDAL) ParentState(ts *types.TipSet) (*state.StateTree, error) {
	panic("not impl")
}
func (dal *MockDAL) ParentStateTsk(tsk types.TipSetKey) (*state.StateTree, error) {
	panic("not impl")
}
func (dal *MockDAL) StateTree(st cid.Cid) (*state.StateTree, error) {
	panic("not impl")
}
func (dal *MockDAL) LoadActor(ctx context.Context, addr address.Address, ts *types.TipSet) (*types.Actor, error) {
	panic("not impl")
}
func (dal *MockDAL) LoadActorTsk(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	panic("not impl")
}
func (dal *MockDAL) LoadActorRaw(_ context.Context, addr address.Address, st cid.Cid) (*types.Actor, error) {
	panic("not impl")
}
func (dal *MockDAL) GetNtwkVersion(ctx context.Context, height abi.ChainEpoch) network.Version {
	panic("not impl")
}
func (dal *MockDAL) GetVMCirculatingSupplyDetailed(ctx context.Context, height abi.ChainEpoch, st *state.StateTree) (api.CirculatingSupply, error) {
	panic("not impl")
}
func (dal *MockDAL) AddEnum(ctx context.Context, ns string, entry ...string) error {
	panic("not impl")
}
func (dal *MockDAL) LookupEnum(ctx context.Context, ns string, entry string) (int, error) {
	panic("not impl")
}
func (dal *MockDAL) ActorStore(ctx context.Context) adt.Store {
	return store.ActorStore(ctx, dal.blockStore)
}
func (dal *MockDAL) LoadTipSet(tsk types.TipSetKey) (*types.TipSet, error) {
	panic("not impl")
}
func (dal *MockDAL) Weight(ctx context.Context, ts *types.TipSet) (types.BigInt, error) {
	panic("not impl")
}
func (dal *MockDAL) GetGenesis() (*types.BlockHeader, error) {
	panic("not impl")
}
func (dal *MockDAL) ChainBlockstore() bstore.Blockstore {
	panic("not impl")
}

func Test_extractClaimedPowerV6(t *testing.T) {
	ctx := context.Background()
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()
	mockDAL := &MockDAL{localBs}
	res := extract.NewRes(0, 0)
	ectx, err := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, extract.DryOptions())
	require.NoError(t, err)
	claimsCid, _ := cid.Decode("bafy2bzaceafax6er2dups5gg2fpb3xjcicux2koydgwpaut3vctpihwfylrqg")
	headCid, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	err = extractClaimedPowerV6(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headCid},
		Epoch: abi.ChainEpoch(455000),
	}, &power6.State{Claims: claimsCid})

	require.NoError(t, err)
}

func GeneratePowerData(t *testing.T) {
	// Generate Data for:
	// 		Network: calibration
	// 		Epoch: 455000
	// 		ActorAddress: t04
	// 		Head: bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay
	// 		Claims: bafy2bzaceafax6er2dups5gg2fpb3xjcicux2koydgwpaut3vctpihwfylrqg

	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	rootCid, _ := cid.Decode("bafy2bzacedb7yvsktcclo3no4kl2hyex5gwaoog5wjw76gk6kudfjflmonnay")
	rpcBS, err := testutils.NewApiBlockStore(ctx, url)
	require.NoError(t, err)
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
	require.NoError(t, err)
}
