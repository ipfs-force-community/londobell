package actor

import (
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

var testBuilder = cid.V1Builder{Codec: cid.Raw, MhType: mh.IDENTITY}
