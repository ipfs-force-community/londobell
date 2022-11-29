package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type StateLookupIDReq struct {
	Epoch   int64  `json:"epoch"`
	Addr    string `json:"addr"`
	Reverse bool   `json:"reverse"` // 默认false
}

type StateLookupIDRes struct {
	Epoch      abi.ChainEpoch  `json:"epoch"`
	Addr       address.Address `json:"addr"`
	LookupAddr address.Address `json:"lookup_addr"`
}
