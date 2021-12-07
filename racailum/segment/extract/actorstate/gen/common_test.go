package gen

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	bstore "github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type MockDAL struct {
	mock.Mock
}

func (m *MockDAL) ExecutionTrace(ctx context.Context, ts *types.TipSet) (cid.Cid, []*api.InvocResult, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(cid.Cid), args.Get(1).([]*api.InvocResult), args.Error(2)
}
func (m *MockDAL) ParentState(ts *types.TipSet) (*state.StateTree, error) {
	args := m.Called(ts)
	return args.Get(0).(*state.StateTree), args.Error(1)
}
func (m *MockDAL) ParentStateTsk(tsk types.TipSetKey) (*state.StateTree, error) {
	args := m.Called(tsk)
	return args.Get(0).(*state.StateTree), args.Error(1)
}
func (m *MockDAL) StateTree(st cid.Cid) (*state.StateTree, error) {
	args := m.Called(st)
	return args.Get(0).(*state.StateTree), args.Error(1)
}
func (m *MockDAL) LoadActor(ctx context.Context, addr address.Address, ts *types.TipSet) (*types.Actor, error) {
	args := m.Called(ctx, addr, ts)
	return args.Get(0).(*types.Actor), args.Error(1)
}
func (m *MockDAL) LoadActorTsk(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	args := m.Called(ctx, addr, tsk)
	return args.Get(0).(*types.Actor), args.Error(1)
}
func (m *MockDAL) LoadActorRaw(ctx context.Context, addr address.Address, st cid.Cid) (*types.Actor, error) {
	args := m.Called(ctx, addr, st)
	return args.Get(0).(*types.Actor), args.Error(1)
}
func (m *MockDAL) GetNtwkVersion(ctx context.Context, height abi.ChainEpoch) network.Version {
	args := m.Called(ctx, height)
	return args.Get(0).(network.Version)
}
func (m *MockDAL) GetVMCirculatingSupplyDetailed(ctx context.Context, height abi.ChainEpoch, st *state.StateTree) (api.CirculatingSupply, error) {
	args := m.Called(ctx, height, st)
	return args.Get(0).(api.CirculatingSupply), args.Error(1)
}
func (m *MockDAL) AddEnum(ctx context.Context, ns string, entry ...string) error {
	args := m.Called(ctx, ns, entry)
	return args.Error(0)
}
func (m *MockDAL) LookupEnum(ctx context.Context, ns string, entry string) (int, error) {
	args := m.Called(ctx, ns, entry)
	return args.Int(0), args.Error(1)
}
func (m *MockDAL) ActorStore(ctx context.Context) adt.Store {
	args := m.Called(ctx)
	return args.Get(0).(adt.Store)
}
func (m *MockDAL) LoadTipSet(tsk types.TipSetKey) (*types.TipSet, error) {
	args := m.Called(tsk)
	return args.Get(0).(*types.TipSet), args.Error(1)
}
func (m *MockDAL) Weight(ctx context.Context, ts *types.TipSet) (types.BigInt, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(types.BigInt), args.Error(1)
}
func (m *MockDAL) GetGenesis() (*types.BlockHeader, error) {
	args := m.Called()
	return args.Get(0).(*types.BlockHeader), args.Error(1)
}
func (m *MockDAL) ChainBlockstore() bstore.Blockstore {
	args := m.Called()
	return args.Get(0).(bstore.Blockstore)
}
