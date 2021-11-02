package actorstate

import (
	"bytes"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/specs-actors/actors/builtin"
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

func NormalEpochRange(head *common.ActorHead) []abi.ChainEpoch {
	return []abi.ChainEpoch{head.Epoch + builtin.EpochsInDay, head.Epoch + builtin.EpochsInDay*7,
		head.Epoch + builtin.EpochsInDay*14, head.Epoch + builtin.EpochsInDay*30}
}
