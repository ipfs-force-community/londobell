package model

import "github.com/filecoin-project/go-state-types/abi"

type ActorStateRes struct {
	ActorID string
	Code    string
	Balance string
	Epoch   abi.ChainEpoch
}
