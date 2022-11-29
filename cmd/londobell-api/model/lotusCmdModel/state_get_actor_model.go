package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
)

type StateGetActorReq struct {
	Epoch int64  `json:"epoch"`
	Addr  string `json:"addr"`
}

type StateGetActorRes struct {
	Epoch abi.ChainEpoch  `json:"epoch"`
	Addr  address.Address `json:"addr"`
	Actor *types.Actor    `json:"actor"`
	Type  string          `json:"type"`
}
