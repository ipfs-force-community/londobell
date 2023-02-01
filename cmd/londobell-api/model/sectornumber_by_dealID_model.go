package model

import "github.com/filecoin-project/go-state-types/abi"

type SectorNumberByDealIDReq struct {
	Miner  string `json:"miner"`
	DealID uint64 `json:"dealID"`
}

type SectorNumberByDealIDRes struct {
	Miner        string
	DealID       abi.DealID
	SectorNumber abi.SectorNumber
}
