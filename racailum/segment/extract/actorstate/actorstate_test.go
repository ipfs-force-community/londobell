package actorstate

import (
	"testing"

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
