package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/dline"
)

type StateMinerProvingDeadlineReq struct {
	Miner string `json:"miner"`
	Epoch int64  `json:"epoch"`
}

type StateMinerProvingDeadlineRes struct {
	Miner        address.Address `json:"miner"`
	Epoch        abi.ChainEpoch  `json:"epoch"`
	DeadlineInfo *dline.Info     `json:"deadline_info"`
}
