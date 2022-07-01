package model

type SectorPowerReq struct {
	Miner  string `json:"miner"`
	Epoch  int64  `json:"epoch"`
	Sector uint64 `json:"sector"`
}

type SectorPowerRes SectorRes
