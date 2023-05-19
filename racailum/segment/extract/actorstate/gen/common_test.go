package gen

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/api"
	bstore "github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/testutils"
)

type MockDAL struct {
	mock.Mock
}

func (m *MockDAL) TipSetState(ctx context.Context, ts *types.TipSet) (st cid.Cid, rec cid.Cid, err error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(cid.Cid), args.Get(1).(cid.Cid), args.Error(2)
}

func (m *MockDAL) ExecutionTrace(ctx context.Context, ts *types.TipSet) (cid.Cid, []*api.InvocResult, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(cid.Cid), args.Get(1).([]*api.InvocResult), args.Error(2)
}
func (m *MockDAL) ParentState(ts *types.TipSet) (*state.StateTree, error) {
	args := m.Called(ts)
	return args.Get(0).(*state.StateTree), args.Error(1)
}
func (m *MockDAL) ParentStateTsk(ctx context.Context, tsk types.TipSetKey) (*state.StateTree, error) {
	args := m.Called(ctx, tsk)
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
func (m *MockDAL) GetNetworkVersion(ctx context.Context, height abi.ChainEpoch) network.Version {
	args := m.Called(ctx, height)
	return args.Get(0).(network.Version)
}
func (m *MockDAL) GetVMCirculatingSupplyDetailed(ctx context.Context, height abi.ChainEpoch, st *state.StateTree) (api.CirculatingSupply, error) {
	args := m.Called(ctx, height, st)
	return args.Get(0).(api.CirculatingSupply), args.Error(1)
}
func (m *MockDAL) SearchForMessage(ctx context.Context, head *types.TipSet, mcid cid.Cid, lookbackLimit abi.ChainEpoch, allowReplaced bool) (*types.TipSet, *types.MessageReceipt, cid.Cid, error) {
	args := m.Called(ctx, head, mcid, lookbackLimit, allowReplaced)
	return args.Get(0).(*types.TipSet), args.Get(1).(*types.MessageReceipt), args.Get(2).(cid.Cid), args.Error(3)
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
func (m *MockDAL) LoadTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	args := m.Called(ctx, tsk)
	return args.Get(0).(*types.TipSet), args.Error(1)
}
func (m *MockDAL) Weight(ctx context.Context, ts *types.TipSet) (types.BigInt, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(types.BigInt), args.Error(1)
}
func (m *MockDAL) GetGenesis(ctx context.Context) (*types.BlockHeader, error) {
	args := m.Called(ctx)
	return args.Get(0).(*types.BlockHeader), args.Error(1)
}
func (m *MockDAL) ChainBlockstore() bstore.Blockstore {
	args := m.Called()
	return args.Get(0).(bstore.Blockstore)
}
func (m *MockDAL) MessagesForBlock(ctx context.Context, b *types.BlockHeader) ([]*types.Message, []*types.SignedMessage, error) {
	args := m.Called(ctx, b)
	return args.Get(0).([]*types.Message), args.Get(1).([]*types.SignedMessage), args.Error(2)
}
func (m *MockDAL) ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(abi.TokenAmount), args.Error(1)
}

func (m *MockDAL) LookupID(ctx context.Context, addr address.Address, ts *types.TipSet) (address.Address, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(address.Address), args.Error(1)
}

/*
Network: calibration
Epoch: 792000
ActorAddress: t04
Head: bafy2bzaceavqgbrf6tasxwsh33efuy5wepadh3bodjacseqdex2qpn7tlh5fk
*/
const testPowerActorCid = "bafy2bzaceavqgbrf6tasxwsh33efuy5wepadh3bodjacseqdex2qpn7tlh5fk"

/*
Network: calibration
Epoch: 792000
ActorAddress: t05
Head: bafy2bzacebzk4bvrrzhyvzm2gfjymwk7yatpzr56uiwkdgfyzg4evpvgcihiw
*/
const testMarketActorCid = "bafy2bzacebzk4bvrrzhyvzm2gfjymwk7yatpzr56uiwkdgfyzg4evpvgcihiw"

/*
Network: calibration
Epoch: 792000
ActorAddress: t02
Head: bafy2bzacebwol6ndjdyw7bwfvtctydbc32cik6xkm4wousgg7erejhogakpjq
*/
const testRewardActorCid = "bafy2bzacebwol6ndjdyw7bwfvtctydbc32cik6xkm4wousgg7erejhogakpjq"

/*
Network: calibration
Epoch: 792000
ActorAddress: t011092
Head: bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a
*/
const testMultisigActorCid = "bafy2bzaceaa3zevs2vazynsd5fjolq2sc2633hj6poly2xvscpc2n56cook4a"

/*
Network: calibration
Epoch: 792000
ActorAddress: t06
Head: bafy2bzaceajlkb4izlku6fosrdqm4h7vzxyrqveivmosta25appren7uqfsby
*/
const testVerifRegActorCid = "bafy2bzaceajlkb4izlku6fosrdqm4h7vzxyrqveivmosta25appren7uqfsby"

/*
Network: calibration
Epoch: 792000
ActorAddress: t031582
Head: bafy2bzaceauab37rktpgup3x6sh3rg5n37ddoxjck5r5gukmdqrqsipco7bm4
*/
const testMinerSectorActorCid = "bafy2bzacebtzupwrfw2fgmfqzafht4imbhbks2fkaqfs77ay2cc63au3jap3c"

/*
Network: calibration
Epoch: 792000
ActorAddress: t027502
Head: bafy2bzacedmi47gmsb4wyhlpctizrv4bedex2frpi47leufximpkx37kvmrvk
*/
const testPendingTxnsActorCid = "bafy2bzacedmi47gmsb4wyhlpctizrv4bedex2frpi47leufximpkx37kvmrvk"

func GenerateLocalData(t *testing.T) {
	url := os.Getenv("TEST_LOTUS_URL")
	ctx := context.Background()
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	for _, c := range []string{testPowerActorCid, testMarketActorCid, testRewardActorCid, testMultisigActorCid, testVerifRegActorCid, testMinerSectorActorCid, testPendingTxnsActorCid} {
		rootCid, _ := cid.Decode(c)
		rpcBS, err := testutils.NewApiBlockStore(ctx, url)
		require.NoError(t, err)
		err = testutils.GenerateFullTree(ctx, rootCid, rpcBS, localBS)
		require.NoError(t, err)
	}
}
