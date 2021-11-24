package main

import (
	"context"
	"fmt"
	"github.com/dtynn/londobell/common"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/rand"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/metrics"
	"github.com/ipfs/go-cid"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
)

type messageFinder struct {
	mcid cid.Cid // the message cid to find
	outm *types.Message
	outr *vm.ApplyRet
}

func replay(ctx context.Context, sm common.StateManager, ts *types.TipSet, msglist []types.Message) (invocResult []*api.InvocResult, err error) {
	for _, msg := range msglist {
		mstToReplay := msg.Cid()
		fmt.Println(mstToReplay.String())

		//m, r, err := sm.(*stmgr.StateManager).Replay(ctx, ts, mstToReplay)  //finder.outr == nil时，应用不属于tipset的消息
		//if err != nil {
		//	if xerrors.Is(err, xerrors.Errorf("unexpected error during execution: %w", err)) {
		//		//执行不属于tipset的消息
		//		m, r, err = replayCustom(ctx, msg, sm, ts)
		//		if err != nil {
		//			return nil, err
		//		}
		//	} else {
		//		return nil, err
		//	}
		//}

		m, r, err := replayCustom(ctx, msg, sm, ts)
		if err != nil {
			return nil, err
		}

		var errstr string
		if r.ActorErr != nil {
			errstr = r.ActorErr.Error()
		}
		invocResult = append(invocResult, &api.InvocResult{ //只要ExecutionTrace和GasCost
			MsgCid:         mstToReplay,
			Msg:            m,
			MsgRct:         &r.MessageReceipt,
			GasCost:        stmgr.MakeMsgGasCost(m, r),
			ExecutionTrace: r.ExecutionTrace,
			Error:          errstr,
			Duration:       r.Duration,
		})
	}

	return invocResult, nil
}

func replayCustom(ctx context.Context, msg types.Message, sm common.StateManager, ts *types.TipSet) (*types.Message, *vm.ApplyRet, error){
	//var finder messageFinder
	//finder.mcid = msgcid

	m, ret, err := executeTipsetCustom(ctx, &msg, sm, ts)
	if err != nil {
		return nil, nil, err
	}

	return m, ret, nil
}

func executeTipsetCustom(ctx context.Context, cmsg types.ChainMsg, sm common.StateManager, ts *types.TipSet) (*types.Message, *vm.ApplyRet, error) {
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

func applyBlocksCustom(ctx context.Context, cmsg types.ChainMsg, sm common.StateManager, pstate cid.Cid, epoch abi.ChainEpoch, r vm.Rand, baseFee abi.TokenAmount, ts *types.TipSet) (*types.Message, *vm.ApplyRet, error) {
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

	//if em != nil {
	//	m := cmsg.VMMessage()
	//	if err := em.MessageApplied(ctx, ts, cmsg.Cid(), m, ret, false); err != nil {
	//		return err
	//	}
	//}


	//支持隐式消息？？

}

//根据消息cid解码ChainMsg类型cm，调用vmi.ApplyMessage(ctx, cm)，得到ApplyRet类型
//其中vmi由makeVmWithBaseState(pstate)得到，而pstate为blockheader的ParentStateRoot;由mcid推出blockheader??直接使用tipset的blockheader

//应用显示消息和隐式消息，怎样算应用成功？
//不会改变状态？？








