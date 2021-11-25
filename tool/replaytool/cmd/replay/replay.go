package main

import (
	"context"
	"github.com/dtynn/londobell/common"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/rand"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/metrics"
	"github.com/ipfs/go-cid"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
)

func replay(ctx context.Context, sm common.StateManager, ts *types.TipSet, msglist []types.Message) (replayResult []*api.InvocResult, err error) {
	statetree, err := statetreeForTipset(ts, sm)
	if err != nil {
		return nil, err
	}

	for _, msg := range msglist {
		maxNonce, err := nonceForActor(statetree, msg)
		if err != nil {
			return nil, err
		}

		msg.Nonce = maxNonce
		mstToReplay := msg.Cid()

		m, r, err := replayCustom(ctx, &msg, sm, ts)
		if err != nil {
			return nil, err
		}

		var errstr string
		if r.ActorErr != nil {
			errstr = r.ActorErr.Error()
		}
		replayResult = append(replayResult, &api.InvocResult{
			MsgCid:         mstToReplay,
			Msg:            m,
			MsgRct:         &r.MessageReceipt,
			GasCost:        stmgr.MakeMsgGasCost(m, r),
			ExecutionTrace: r.ExecutionTrace,
			Error:          errstr,
			Duration:       r.Duration,
		})
	}

	return replayResult, nil
}

func statetreeForTipset(ts *types.TipSet, sm common.StateManager) (*state.StateTree, error) {
	statetree, err := sm.ParentState(ts)
	if err != nil {
		return nil, err
	}

	return statetree, nil
}

func nonceForActor(statetree *state.StateTree, msg types.Message) (uint64, error) {
	act, err := statetree.GetActor(msg.From) //没有这个actor？？
	if err != nil {
		return 0, err
	}

	return act.Nonce, nil
}

func replayCustom(ctx context.Context, msg *types.Message, sm common.StateManager, ts *types.TipSet) (*types.Message, *vm.ApplyRet, error) {
	m, ret, err := executeTipsetCustom(ctx, msg, sm, ts)
	if err != nil {
		return nil, nil, err
	}

	return m, ret, nil
}

func executeTipsetCustom(ctx context.Context, cmsg *types.Message, sm common.StateManager, ts *types.TipSet) (*types.Message, *vm.ApplyRet, error) {
	ctx, span := trace.StartSpan(ctx, "computeTipSetState")
	defer span.End()

	blks := ts.Blocks()

	//不需要判断duplicate miner in a tipset？？

	pstate := blks[0].ParentStateRoot
	r := rand.NewStateRand(sm.(*stmgr.StateManager).ChainStore(), ts.Cids(), sm.(*stmgr.StateManager).Beacon())
	baseFee := blks[0].ParentBaseFee
	height := blks[0].Height

	return applyBlocksCustom(ctx, cmsg, sm, pstate, height, r, baseFee, ts)
}

func applyBlocksCustom(ctx context.Context, cmsg *types.Message, sm common.StateManager, pstate cid.Cid, epoch abi.ChainEpoch, r vm.Rand, baseFee abi.TokenAmount, ts *types.TipSet) (*types.Message, *vm.ApplyRet, error) {
	done := metrics.Timer(ctx, metrics.VMApplyBlocksTotal)
	defer done()

	partDone := metrics.Timer(ctx, metrics.VMApplyEarly)
	defer func() {
		partDone()
	}()

	makeVmWithBaseState := func(base cid.Cid) (*vm.VM, error) {
		vmopt := &vm.VMOpts{
			StateBase:      base,
			Epoch:          epoch,
			Rand:           r,
			Bstore:         sm.(*stmgr.StateManager).ChainStore().StateBlockstore(),
			Actors:         filcns.NewActorRegistry(),
			Syscalls:       sm.(*stmgr.StateManager).VMSys(),
			CircSupplyCalc: sm.(*stmgr.StateManager).GetVMCirculatingSupply,
			NtwkVersion:    sm.GetNtwkVersion,
			BaseFee:        baseFee,
			LookbackState:  stmgr.LookbackStateGetterForTipset(sm.(*stmgr.StateManager), ts),
		}

		return sm.(*stmgr.StateManager).VMConstructor()(ctx, vmopt)
	}

	vmi, err := makeVmWithBaseState(pstate)
	if err != nil {
		return nil, nil, xerrors.Errorf("making vm: %w", err)
	}

	m := cmsg.VMMessage()
	ret, err := vmi.ApplyMessage(ctx, cmsg)
	if err != nil {
		return nil, nil, err
	}

	return m, ret, nil
}
