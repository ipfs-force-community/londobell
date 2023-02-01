package model

import "github.com/filecoin-project/go-state-types/abi"

type BalanceRes struct {
	Actor   string `bson:"Addr" json:"Actor"`
	Epoch   abi.ChainEpoch
	Balance string
	Type    string `bson:"Code" json:"Type"`
}
