package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type DetailForDealRes struct {
	DealID               int64
	Epoch                abi.ChainEpoch
	Cid                  string
	PieceCID             string
	VerifiedDeal         bool
	Client               string
	Provider             string
	ProviderCollateral   string
	ClientCollateral     string
	StartEpoch           abi.ChainEpoch
	EndEpoch             abi.ChainEpoch
	PieceSize            uint64
	StoragePricePerEpoch string
}
