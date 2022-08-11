package model

import (
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type SectorReq struct {
	Miner string `json:"miner"`
	Epoch int64  `json:"epoch"`
}

type SectorRes struct {
	Miner                 address.Address  `json:"miner"`
	Date                  time.Time        `json:"date"`
	SectorNumber          abi.SectorNumber `json:"sector_number"`
	Version               string           `json:"version"`
	Size                  abi.SectorSize   `json:"size"`
	Activation            abi.ChainEpoch   `json:"activation"`
	Expiration            abi.ChainEpoch   `json:"expiration"`
	Pledge                abi.TokenAmount  `json:"pledge"`
	DealWeight            abi.DealWeight   `json:"deal_weight"`
	VerifiedDealWeight    abi.DealWeight   `json:"verified_deal_weight"`
	ExpectedDayReward     abi.TokenAmount  `json:"expected_day_reward"`
	ExpectedStoragePledge abi.TokenAmount  `json:"expected_storage_pledge"`
	ReplaceSectorAge      abi.ChainEpoch   `json:"replaced_sector_age"`
	ReplaceDayReward      abi.TokenAmount  `json:"replaced_day_reward"`
}
