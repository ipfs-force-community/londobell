package model

import "github.com/filecoin-project/go-state-types/abi"

type SectorPowerReq struct {
	Miner  string `json:"miner"`
	Epoch  int64  `json:"epoch"`
	Sector uint64 `json:"sector"`
}

type SectorPowerRes struct {
	Miner           string           `json:"miner"`
	Epoch           int64            `json:"epoch"`
	Sector          uint64           `json:"sector"`
	QualityAdjPower abi.StoragePower `json:"quality_adj_power"`
}
