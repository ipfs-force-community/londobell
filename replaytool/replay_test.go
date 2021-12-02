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

func benchMarkReplay(n int, b *testing.B) {
	cg, err := gen.NewGenerator()
	if err != nil {
		b.Fatal(err)
	}

	signedMsglists := make([]*types.SignedMessage, 0, 20*n)
	for i := 0; i < n; i++ {
		signedMsglist, _ := cg.GetMessages(cg)
		signedMsglists = append(signedMsglists, signedMsglist...)
	}

	msglist := make([]types.Message, 0, len(signedMsglists))
	for i := range signedMsglists {
		msglist = append(msglist, signedMsglists[i].Message)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err = Replay(context.TODO(), cg.StateManager(), cg.CurTipset.TipSet(), msglist, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplay20times(b *testing.B) {
	benchMarkReplay(1, b)
}

func BenchmarkReplay40times(b *testing.B) {
	benchMarkReplay(2, b)
}

func BenchmarkReplay60times(b *testing.B) {
	benchMarkReplay(3, b)
}
