package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type MarketBalanceRes struct {
	Actor         address.Address `json:"actor"`
	Epoch         abi.ChainEpoch  `json:"epoch"`
	EscrowBalance abi.TokenAmount `json:"escrow_balance"`
	LockedBalance abi.TokenAmount `json:"locked_balance"`
}
