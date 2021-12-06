package replaytool

import (
	"context"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/ipfs/go-cid"
)

type InvocationTracer struct {
	trace *[]*api.InvocResult
	mcids []cid.Cid
}

func (i *InvocationTracer) MessageApplied(ctx context.Context, ts *types.TipSet, mcid cid.Cid, msg *types.Message, ret *vm.ApplyRet, implicit bool) error {
	flag := false
	for _, cid := range i.mcids {
		if cid == mcid {
			flag = true
			break
		}
	}

	if !flag {
		return nil
	}

	ir := &api.InvocResult{
		MsgCid:         mcid,
		Msg:            msg,
		MsgRct:         &ret.MessageReceipt,
		ExecutionTrace: ret.ExecutionTrace,
		Duration:       ret.Duration,
	}
	if ret.ActorErr != nil {
		ir.Error = ret.ActorErr.Error()
	}
	if ret.GasCosts != nil {
		ir.GasCost = stmgr.MakeMsgGasCost(msg, ret)
	}
	*i.trace = append(*i.trace, ir)

	return nil
}
