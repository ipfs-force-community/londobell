package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/rand"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs/go-cid"
)

func replay(ctx context.Context, sm common.StateManager, ts *types.TipSet, msglist []types.Message, filepath string) error {
	err := applyMessages(ctx, sm, ts, msglist, filepath)
	if err != nil {
		return err
	}

	return nil
}

func applyMessages(ctx context.Context, sm common.StateManager, ts *types.TipSet, msglist []types.Message, filepath string) error {
	blks := ts.Blocks()

	pstate := blks[0].ParentStateRoot
	r := rand.NewStateRand(sm.(*stmgr.StateManager).ChainStore(), ts.Cids(), sm.(*stmgr.StateManager).Beacon())
	baseFee := blks[0].ParentBaseFee
	epoch := blks[0].Height

	makeVMWithBaseState := func(base cid.Cid) (*vm.VM, error) {
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

	vmi, err := makeVMWithBaseState(pstate)
	if err != nil {
		return err
	}

	nonceMap := make(map[address.Address]uint64)

	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	defer f.Close()

	for i, msg := range msglist {
		if _, ok := nonceMap[msg.From]; !ok {
			curNonce, err := nonceForActor(ts, sm, msg)
			if err != nil {
				return err
			}

			msg.Nonce = curNonce
			nonceMap[msg.From] = curNonce + 1
		} else {
			msg.Nonce = nonceMap[msg.From]
			nonceMap[msg.From]++
		}

		msglist[i] = msg
		r, err := vmi.ApplyMessage(ctx, &msglist[i])
		if err != nil {
			return err
		}

		var errstr string
		if r.ActorErr != nil {
			errstr = r.ActorErr.Error()
		}

		mstToReplay := msg.Cid()
		m := msg.VMMessage()

		result := &api.InvocResult{
			MsgCid:         mstToReplay,
			Msg:            m,
			MsgRct:         &r.MessageReceipt,
			GasCost:        stmgr.MakeMsgGasCost(m, r),
			ExecutionTrace: r.ExecutionTrace,
			Error:          errstr,
			Duration:       r.Duration,
		}

		fileContent, err := json.Marshal(result)
		if err != nil {
			return err
		}

		_, err = f.Write(fileContent)
		if err != nil {
			return err
		}

		_, err = f.WriteString("\n") // 其他系统 todo
		if err != nil {
			return err
		}
	}

	return nil
}

func nonceForActor(ts *types.TipSet, sm common.StateManager, msg types.Message) (uint64, error) {
	statetree, err := sm.ParentState(ts)
	if err != nil {
		return 0, err
	}

	act, err := statetree.GetActor(msg.From)
	if err != nil {
		return 0, err
	}

	return act.Nonce, nil
}
