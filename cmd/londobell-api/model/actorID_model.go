package model

import (
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type ActorIDReq struct {
	Epoch int64 `json:"epoch"`
}

type ActorIDRes struct {
	ActorIDs  []address.Address `json:"actor_ids"`
	Epoch     abi.ChainEpoch    `json:"epoch"`
	BlockTime time.Time         `json:"block_time"`
}
