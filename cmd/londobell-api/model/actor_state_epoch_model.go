package model

import "github.com/filecoin-project/go-state-types/abi"

type ActorStateEpochRes struct {
	Addr    string
	Code    string
	Balance string
	Epoch   abi.ChainEpoch
	Detail  interface{} //state different from various actors
}
