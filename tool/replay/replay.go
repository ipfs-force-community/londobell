package replay

import (
	"context"
	"github.com/dtynn/londobell/common"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

func Replay(ctx context.Context, sm common.StateManager, ts *types.TipSet, msgcids []cid.Cid) (invocResult []*api.InvocResult, err error) {
	for _, msgcid := range msgcids {
		mstToReplay := msgcid

		m, r, err := sm.Replay(ctx, ts, mstToReplay)
		if err != nil {
			return nil, err
		}

		var errstr string
		if r.ActorErr != nil {
			errstr = r.ActorErr.Error()
		}
		invocResult = append(invocResult, &api.InvocResult{
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