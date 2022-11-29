package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
)

type StateReadStateReq struct {
	Epoch int64  `json:"epoch"`
	Actor string `json:"actor"`
}

type StateReadStateRes struct {
	Epoch      abi.ChainEpoch  `json:"epoch"`
	Actor      address.Address `json:"actor"`
	ActorState *api.ActorState `json:"actor_state"`
}
