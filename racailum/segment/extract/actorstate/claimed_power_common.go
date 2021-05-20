package actorstate

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
)

func genClaimedPowerID(buf *bytes.Buffer, keystr string, detail cbor.Er) (cid.Cid, error) {
	if _, err := buf.WriteString(keystr); err != nil {
		return cid.Undef, fmt.Errorf("write key string %s: %w", keystr, err)
	}

	if err := detail.MarshalCBOR(buf); err != nil {
		return cid.Undef, fmt.Errorf("write claim data for %s: %w", keystr, err)
	}

	id, err := common.CidBuilder.Sum(buf.Bytes())
	if err != nil {
		return cid.Undef, fmt.Errorf("construct cid for %s: %w", keystr, err)
	}

	return id, nil
}
