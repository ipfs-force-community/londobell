package model

import "github.com/filecoin-project/go-state-types/abi"

type FinalHeightRes struct {
	Epoch abi.ChainEpoch `bson:"epoch" json:"Epoch"`
}
