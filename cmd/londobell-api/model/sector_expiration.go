package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type SectorExpirationRes struct {
	SectorExpirations []SectorOnChainInfo
}

type SectorInfoRes struct {
	SectorExpirationRes
	QAPowerRes
}

type SectorOnChainInfo struct {
	Expiration         abi.ChainEpoch
	Activation         abi.ChainEpoch
	DealWeight         abi.DealWeight
	VerifiedDealWeight abi.DealWeight
	InitialPledge      abi.TokenAmount
}
