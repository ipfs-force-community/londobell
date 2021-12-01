package replaytool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/api"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/rand"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/common"
)

func Replay(ctx context.Context, sm common.StateManager, ts *types.TipSet, msglist []types.Message, filepath string) ([]*api.InvocResult, error) {
	tsExexc := filcns.NewTipSetExecutor()
	blks := ts.Blocks()

	var parentEpoch abi.ChainEpoch
	pstate := blks[0].ParentStateRoot
	if blks[0].Height > 0 {
		parent, err := sm.(*stmgr.StateManager).ChainStore().GetBlock(blks[0].Parents[0])
		if err != nil {
			return nil, err
		}

		parentEpoch = parent.Height
	}

	r := rand.NewStateRand(sm.(*stmgr.StateManager).ChainStore(), ts.Cids(), sm.(*stmgr.StateManager).Beacon())
	baseFee := blks[0].ParentBaseFee

	nonceMap := make(map[address.Address]uint64)
	fbmsgs := make([]filcns.FilecoinBlockMessages, 1)
	blockMessages := make([]store.BlockMessages, 1)
	blockMessages[0].BlsMessages = make([]types.ChainMsg, len(msglist))
	cidSlice := make([]cid.Cid, len(msglist))

	for i, msg := range msglist {
		if _, ok := nonceMap[msg.From]; !ok {
			curNonce, err := nonceForActor(ts, sm, msg)
			if err != nil {
				return nil, err
			}

			msg.Nonce = curNonce
			nonceMap[msg.From] = curNonce + 1
		} else {
			msg.Nonce = nonceMap[msg.From]
			nonceMap[msg.From]++
		}

		msglist[i] = msg
		blockMessages[0].BlsMessages[i] = &msglist[i]
		cidSlice = append(cidSlice, msglist[i].Cid())
	}

	fbmsgs[0].BlockMessages = blockMessages[0]
	fbmsgs[0].Miner = blks[0].Miner
	fbmsgs[0].WinCount = blks[0].ElectionProof.WinCount

	var invocTrace []*api.InvocResult
	em := InvocationTracer{
		&invocTrace,
		cidSlice,
	}

	_, _, err := tsExexc.ApplyBlocks(ctx, sm.(*stmgr.StateManager), parentEpoch, pstate, fbmsgs, blks[0].Height, r, &em, baseFee, ts)
	if err != nil {
		fmt.Printf("applyblock err:%v", err)
		return nil, err
	}

	if filepath == "" {
		return *em.trace, nil
	}

	fileContent, err := json.Marshal(*em.trace)
	if err != nil {
		return nil, err
	}

	err = common.WriteTofile(filepath, fileContent)
	if err != nil {
		return nil, err
	}

	return *em.trace, nil
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
