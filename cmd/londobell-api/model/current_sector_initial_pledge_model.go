package model

import "github.com/filecoin-project/go-state-types/abi"

type CurrentSectorInitialPledgeRes struct {
	CirculatingRate            float64
	CurrentSectorInitialPledge abi.TokenAmount
}
