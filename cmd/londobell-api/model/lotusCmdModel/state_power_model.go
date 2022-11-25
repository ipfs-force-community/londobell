package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
)

type StatePowerReq struct {
	Miner string `json:"miner"`
	Epoch int64  `json:"epoch"`
}

type StatePowerRes struct {
	Miner      address.Address `json:"miner"`
	Epoch      abi.ChainEpoch  `json:"epoch"`
	MinerPower power.Claim     `json:"mp"`
	TotalPower power.Claim     `json:"tp"`
}
