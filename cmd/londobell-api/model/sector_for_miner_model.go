package model

type SectorForMinerReq struct {
	Miner        string `json:"miner"`
	SectorNumber uint64 `json:"sector_number"`
	Epoch        int64  `json:"epoch"`
}
