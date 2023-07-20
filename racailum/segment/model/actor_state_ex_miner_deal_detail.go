package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.Document = (*DealProposalDetail)(nil)

	dealProposalDetailColName    = getColName(DealProposalDetail{})
	dealProposalDetailEpochField = extractEpochFieldName(DealProposalDetail{})
)

// DealProposalDetailDetail contains the details about a set of deals
type MinerDealProposalDetail struct {
	UnVerifiedDealCount    uint64
	UnVerifiedDealEndCount uint64
	VerifiedDealCount      uint64
	VerifiedDealEndCount   uint64
}

// DealProposalDetail is the data model of DealProposals in market states
type DealProposalDetail struct {
	ActorStateExBasic
	Detail MinerDealProposalDetail
}

// CollectionName impl CollectionName
func (d *DealProposalDetail) CollectionName() string {
	return dealProposalDetailColName
}

// EpochField impl common.Document
func (d *DealProposalDetail) EpochField() *string {
	return &dealProposalDetailEpochField
}

// ResetPolicy impl common.Document
func (d *DealProposalDetail) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(dealProposalDetailEpochField, lower, upper), true
}

func (d *DealProposalDetail) IsMutable() bool {
	return false
}
