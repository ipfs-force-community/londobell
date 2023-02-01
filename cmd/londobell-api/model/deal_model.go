package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type Deal struct {
	ID                   int64 `bson:"_id" json:"_id"`
	Epoch                abi.ChainEpoch
	PieceCID             string
	PieceSize            uint64
	VerifiedDeal         bool
	Client               string
	Provider             string
	StartEpoch           abi.ChainEpoch
	EndEpoch             abi.ChainEpoch
	StoragePricePerEpoch string
	ProviderCollateral   string
	ClientCollateral     string
}

type DealsRes struct {
	TotalCount int64  `json:"totalCount"`
	Deals      []Deal `json:"deals"`
}
