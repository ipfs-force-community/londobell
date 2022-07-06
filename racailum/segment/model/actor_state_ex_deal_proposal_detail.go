package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	market8 "github.com/filecoin-project/go-state-types/builtin/v8/market"
	"github.com/filecoin-project/specs-actors/v5/actors/builtin/market"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.Document = (*DealProposal)(nil)

	dealProposalColName      = getColName(DealProposal{})
	dealProposalEpochField   = extractEpochFieldName(DealProposal{})
	dealProposalEpochFieldV8 = extractEpochFieldName(DealProposalV8{})
)

// DealProposal contains the details about a set of deals
type DealProposal struct {
	ID                  int64 `bson:"_id"`
	Epoch               abi.ChainEpoch
	market.DealProposal `bson:",inline"`
}

type DealProposalV8 struct {
	ID                   int64 `bson:"_id"`
	Epoch                abi.ChainEpoch
	market8.DealProposal `bson:",inline"`
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

func (d *DealProposalV8) CollectionName() string {
	return dealProposalColName
}

func (d *DealProposalV8) EpochField() *string {
	return &dealProposalEpochField
}

func (d *DealProposalV8) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(dealProposalEpochFieldV8, lower, upper), true
}

func (d *DealProposalV8) Indexes() [][]string {
	return [][]string{
		[]string{dealProposalEpochFieldV8, "VerifiedDeal"},
		[]string{"VerifiedDeal"},
		[]string{"Provider"},
		[]string{"Client"},
	}
}
