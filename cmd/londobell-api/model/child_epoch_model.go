package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

type ChildEpochRes struct {
	CurrentEpoch  abi.ChainEpoch `bson:"_id"`
	CurrentTipset []cid.Cid
	ChildEpoch    abi.ChainEpoch
	ChildTipset   []cid.Cid
}
