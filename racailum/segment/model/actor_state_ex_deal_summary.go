package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	"github.com/dtynn/londobell/common"
)

var (
	_ common.Document = (*DealProposalSummary)(nil)

	dealProposalSummaryColName    = getColName(DealProposalSummary{})
	dealProposalSummaryEpochField = extractEpochFieldName(DealProposalSummary{})
)

// EmptyDealProposalSummaryDetail returns a empty detail with all fields initialized
func EmptyDealProposalSummaryDetail() DealProposalSummaryDetail {
	return DealProposalSummaryDetail{
		Count:              0,
		PieceSize:          big.Zero(),
		Clients:            0,
		Providers:          0,
		ProviderCollateral: abi.NewTokenAmount(0),
		ClientCollateral:   abi.NewTokenAmount(0),
	}
}

// DealProposalSummaryDetail contains the details about a set of deals
type DealProposalSummaryDetail struct {
	Count     uint64
	PieceSize big.Int
	Clients   uint64
	Providers uint64

	ProviderCollateral abi.TokenAmount
	ClientCollateral   abi.TokenAmount
}

// DealProposalSummary is the data model of DealProposals in market states
type DealProposalSummary struct {
	ActorStateExBasic
	Detail struct {
		Regular  DealProposalSummaryDetail
		Verified DealProposalSummaryDetail
	}
}

// CollectionName impl CollectionName
func (d *DealProposalSummary) CollectionName() string {
	return dealProposalSummaryColName
}

// EpochField impl common.Document
func (d *DealProposalSummary) EpochField() *string {
	return &dealProposalSummaryEpochField
}

// ResetPolicy impl common.Document
func (d *DealProposalSummary) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(dealProposalSummaryEpochField, lower, upper), true
}
