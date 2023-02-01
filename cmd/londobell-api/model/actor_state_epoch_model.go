package model

import "github.com/filecoin-project/go-state-types/abi"

type ActorStateEpochRes struct {
	Actor   string `bson:"Addr" json:"Actor"`
	Type    string `bson:"Code" json:"Type"`
	Balance string
	Epoch   abi.ChainEpoch
	Detail  interface{} //state different from various actors
}
