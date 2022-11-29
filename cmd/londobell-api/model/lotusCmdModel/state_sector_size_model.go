package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type StateSectorSizeReq struct {
	Epoch int64  `json:"epoch"`
	Miner string `json:"miner"`
}

type StateSectorSizeRes struct {
	Epoch      abi.ChainEpoch  `json:"epoch"`
	Miner      address.Address `json:"miner"`
	SectorSize abi.SectorSize  `json:"sector_size"`
}
