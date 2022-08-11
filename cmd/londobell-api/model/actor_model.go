package model

import (
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type ActorReq struct {
	ActorID string `json:"actorId"`
	Epoch   int64  `json:"epoch"`
}

type ActorRes struct {
	ActorID   address.Address `json:"actor_id"`
	ActorAddr string          `json:"actor_addr"`
	Epoch     abi.ChainEpoch  `json:"epoch"`
	BlockTime time.Time       `json:"block_time"`
	ActorType string          `json:"actor_type"`
	Balance   types.BigInt    `json:"balance"`
	Code      cid.Cid         `json:"code"`
	Head      cid.Cid         `json:"head"`
	State     interface{}     `json:"state"`
}
