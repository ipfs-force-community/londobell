package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/lotus/api"
)

type StateMinerInfoReq struct {
	Miner string `json:"miner"`
	Epoch int64  `json:"epoch"`
}

type StateMinerInfoRes struct {
	Miner            address.Address `json:"miner"`
	Epoch            abi.ChainEpoch  `json:"epoch"`
	MinerInfo        api.MinerInfo   `json:"miner_info"`
	AvailableBalance abi.TokenAmount `json:"available_balance"`
	MinerPower       *api.MinerPower `json:"miner_power"`
	DeadlineInfo     *dline.Info     `json:"deadline_info"`
}
