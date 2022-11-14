package actorstate

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	scbor "github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/specs-actors/v8/actors/builtin"
	evm8 "github.com/ipfs-force-community/custom-actors-parsing/external/v8/evm"
	"github.com/ipfs-force-community/londobell/testutils"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/stretchr/testify/require"
)

func TestGetrealcode(t *testing.T) {
	eamActor, _ := cid.Decode("bafk2bzacedwuvyzfaaf6vpxx4lhervvs4qs4ukfqitjxikeemzpec3lbqu5ba")
	embryoActor, _ := cid.Decode("bafk2bzacecau3tohdilfx66pohfqdrngpuqd5oew2j5iv3c7sjlrkcm5npqos")
	evmActor, _ := cid.Decode("bafk2bzacedzg2dsdry6cy5nzfldtqatuopljgdxt5hxdwn2gmuj3fk566bndg")

	getrealcode(eamActor)
	getrealcode(embryoActor)
	getrealcode(evmActor)
}

func TestCodeEqual(t *testing.T) {
	eamActor, _ := cid.Decode("bafk2bzacedwuvyzfaaf6vpxx4lhervvs4qs4ukfqitjxikeemzpec3lbqu5ba")
	eamActor2, _ := cid.Decode("bafk2bzacedwuvyzfaaf6vpxx4lhervvs4qs4ukfqitjxikeemzpec3lbqu5ba")
	equal := eamActor == eamActor2
	fmt.Println(equal)
}

func TestMarketCodeEqual(t *testing.T) {
	mcode := builtin.StorageMarketActorCodeID
	mcode8, _ := cid.Decode("bafk2bzaceddnsy6esfaxzpcczcwo6vjgwd6zqhy7n67mmok5o2zfj3fyaw6ow")
	fmt.Println(mcode)
	fmt.Println(mcode8)
}

func PutState(t *testing.T) {
	ByteCode, _ := cid.Decode("bafk2bzacebdfozypqvzidx6owdew5iotx5qlx6kbuuopruburupx3jjxjrymc")
	ContractState, _ := cid.Decode("bafy2bzaceco5nbg5npqgmqcxmuj3sdv7kqxci5mscjhys6rtcg2qhlzzbto2e")
	var Nonce uint64 = 1

	ctx := context.Background()
	localBS, _, err := testutils.NewLocalBlockStore(ctx)
	cs := cbor.NewCborStore(localBS)
	require.NoError(t, err)
	s1 := &evm8.State{
		ByteCode:      ByteCode,
		ContractState: ContractState,
		Nonce:         Nonce,
	}

	s2 := &evm8.State{
		ByteCode:      ByteCode,
		ContractState: ContractState,
		Nonce:         Nonce,
	}

	h1, err := cs.Put(ctx, s1)
	if err != nil {
		fmt.Println(err)
	}
	h2, err := cs.Put(ctx, s2)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(h1, h2)
}

func TestImplementCbor(t *testing.T) {
	state := &evm8.State{
		ByteCode:      cid.Undef,
		ContractState: cid.Undef,
		Nonce:         1,
	}

	st := reflect.TypeOf(state)
	fmt.Println(st)
	fmt.Println(st.Implements(reflect.TypeOf((*scbor.Er)(nil)).Elem()))
}
