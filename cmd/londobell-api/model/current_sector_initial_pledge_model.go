package model

import "github.com/filecoin-project/go-state-types/abi"

type CurrentSectorInitialPledgeReq struct {
	Epoch           int64  `json:"epoch"`
	QualityAdjPower string `json:"qaPower"`
}

type CurrentSectorInitialPledgeRes struct {
	CirculatingRate            float64
	FilVested                  abi.TokenAmount
	FilMined                   abi.TokenAmount
	FilBurnt                   abi.TokenAmount
	FilLocked                  abi.TokenAmount
	FilCirculating             abi.TokenAmount
	FilReserveDisbursed        abi.TokenAmount
	CurrentSectorInitialPledge abi.TokenAmount
	DailyFee                   abi.TokenAmount
}
