package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
)

type StateSectorReq struct {
	Epoch        int64  `json:"epoch"`
	Miner        string `json:"miner"`
	SectorNumber string `json:"sector_number"`
}

type StateSectorRes struct {
	Epoch        abi.ChainEpoch           `json:"epoch"`
	Miner        address.Address          `json:"miner"`
	SectorNumber abi.SectorNumber         `json:"sector_number"`
	Sector       *miner.SectorOnChainInfo `json:",inline"`
	Partition    *lminer.SectorLocation   `json:",inline"`
}
