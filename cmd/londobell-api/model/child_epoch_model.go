package model

import "github.com/filecoin-project/go-state-types/abi"

type ChildEpochRes struct {
	ID         abi.ChainEpoch `bson:"_id" json:"_id"`
	ChildEpoch abi.ChainEpoch
}
