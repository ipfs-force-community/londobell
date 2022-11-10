package actorstate

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/specs-actors/v8/actors/builtin"
	"github.com/ipfs/go-cid"
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
