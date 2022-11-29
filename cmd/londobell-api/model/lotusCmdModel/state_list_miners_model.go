package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type StateListMinersReq struct {
	Epoch int64 `json:"epoch"`
}

type StateListMinersRes struct {
	Epoch  abi.ChainEpoch    `json:"epoch"`
	Miners []address.Address `json:"miners"`
}
