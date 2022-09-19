package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type ChildEpochRes struct {
	CurrentEpoch  abi.ChainEpoch `bson:"_id"`
	CurrentTipset []string
	ChildEpoch    abi.ChainEpoch
	ChildTipset   []string
}
