package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v5/actors/builtin/market"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.Document = (*DealProposal)(nil)

	dealProposalColName    = getColName(DealProposal{})
	dealProposalEpochField = extractEpochFieldName(DealProposal{})
)

// DealProposal contains the details about a set of deals
type DealProposal struct {
	ID                  int64 `bson:"_id"`
	Epoch               abi.ChainEpoch
	market.DealProposal `bson:",inline"`
}

// CollectionName impl CollectionName
func (d *DealProposal) CollectionName() string {
	return dealProposalColName
}

// EpochField impl common.Document
func (d *DealProposal) EpochField() *string {
	return &dealProposalEpochField
}

// ResetPolicy impl common.Document
func (d *DealProposal) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(dealProposalEpochField, lower, upper), true
}

// Indexes impl common.Indexed
func (d *DealProposal) Indexes() [][]string {
	return [][]string{
		[]string{dealProposalEpochField, "VerifiedDeal"},
		[]string{"VerifiedDeal"},
		[]string{"Provider"},
		[]string{"Client"},
	}
}
