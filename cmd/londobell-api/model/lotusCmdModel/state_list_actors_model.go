package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type StateListActorsReq struct {
	Epoch int64 `json:"epoch"`
}

type StateListActorsRes struct {
	Epoch  abi.ChainEpoch    `json:"epoch"`
	Actors []address.Address `json:"actors"`
}
