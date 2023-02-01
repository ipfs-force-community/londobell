package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

type PendingMessage struct {
	Cid        cid.Cid
	SignedCid  cid.Cid
	Epoch      abi.ChainEpoch
	From       address.Address
	To         address.Address
	Value      abi.TokenAmount
	GasLimit   int64
	GasPremium abi.TokenAmount
	Method     string
}

type PendingMessagesRes struct {
	TotalCount      int64            `json:"totalCount"`
	PendingMessages []PendingMessage `json:"pendingMessages"`
}
