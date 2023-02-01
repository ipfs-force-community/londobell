package model

import "github.com/filecoin-project/go-state-types/abi"

type BalanceRes struct {
	Addr    string
	Epoch   abi.ChainEpoch
	Balance string
	Code    string
}
