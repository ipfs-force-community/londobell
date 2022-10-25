package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type PreCommitDepositToBurnReq struct {
	Epoch  int64    `json:"epoch"`
	Miners []string `json:"miners"`
}

type PreCommitDepositToBurnRes struct {
	Miner         address.Address `json:"miner"`
	Epoch         abi.ChainEpoch  `json:"epoch"`
	DepositToBurn abi.TokenAmount `json:"depositToBurn"`
}
