package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
)

type StateActiveSectorsReq struct {
	Miner string `json:"miner"`
	Epoch int64  `json:"epoch"`
}

type StateActiveSectorsRes struct {
	Miner         address.Address            `json:"miner"`
	Epoch         abi.ChainEpoch             `json:"epoch"`
	ActiveSectors []*miner.SectorOnChainInfo `json:"active_sectors"`
}
