package replaytool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/policy"
	"github.com/filecoin-project/lotus/chain/gen"
	"github.com/filecoin-project/lotus/chain/types"
)

func init() {
	policy.SetConsensusMinerMinPower(abi.NewStoragePower(2048))
	policy.SetMinVerifiedDealSize(abi.NewStoragePower(256))
}

func TestReplay(t *testing.T) {
	cg, err := gen.NewGenerator()
	require.NoError(t, err)

	signedMsglist, _ := cg.GetMessages(cg)
	msglist := make([]types.Message, 0, len(signedMsglist))
	for i := range signedMsglist {
		msglist = append(msglist, signedMsglist[i].Message)
	}

	result, err := Replay(context.TODO(), cg.StateManager(), cg.CurTipset.TipSet(), msglist, nil)
	require.NoError(t, err)
	require.Equal(t, 20, len(result))
}
