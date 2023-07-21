package model

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v5/actors/builtin/market"
	"github.com/ipfs/go-cid"

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
	ProviderID          addr.Address
	ClientID            addr.Address
	market.DealProposal `bson:",inline"`
}

type DealProposalV8 struct {
	ID              int64 `bson:"_id"`
	Epoch           abi.ChainEpoch
	ProviderID      addr.Address
	ClientID        addr.Address
	MDealProposalV8 `bson:",inline"`
}

type MDealProposalV8 struct {
	PieceCID             cid.Cid `checked:"true"`
	PieceSize            abi.PaddedPieceSize
	VerifiedDeal         bool
	Client               addr.Address
	Provider             addr.Address
	Label                []byte
	StartEpoch           abi.ChainEpoch
	EndEpoch             abi.ChainEpoch
	StoragePricePerEpoch abi.TokenAmount
	ProviderCollateral   abi.TokenAmount
	ClientCollateral     abi.TokenAmount
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
		[]string{"VerifiedDeal", "_id"},
		[]string{"ProviderID", "-_id"},
		[]string{"ClientID", "-_id"},
		[]string{"ProviderID", "VerifiedDeal", "-_id"},
		[]string{"ClientID", "VerifiedDeal", "-_id"},
		[]string{dealProposalEpochField},
		[]string{dealProposalEpochField, "_id"},
	}
}

func (d *DealProposal) IsMutable() bool {
	return false
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
		[]string{"VerifiedDeal", "_id"},
		[]string{"ProviderID", "-_id"},
		[]string{"ClientID", "-_id"},
		[]string{"ProviderID", "VerifiedDeal", "-_id"},
		[]string{"ClientID", "VerifiedDeal", "-_id"},
		[]string{dealProposalEpochFieldV8},
		[]string{dealProposalEpochFieldV8, "_id"},
	}
}

func (d *DealProposalV8) IsMutable() bool {
	return false
}
