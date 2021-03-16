package actorstate

import (
	"bytes"

	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
)

func extractCborObject(dal common.DAL, c cid.Cid, out cbor.Er) error {
	blk, err := dal.ChainBlockstore().Get(c)
	if err != nil {
		return err
	}

	return out.UnmarshalCBOR(bytes.NewReader(blk.RawData()))
}
