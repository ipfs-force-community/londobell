package model

import (
	"bytes"
	"fmt"

	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin/market"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.Document = (*NewDealProposal)(nil)
	_ common.Document = (*NewDealProposalV8)(nil)

	newDealProposalColName    = getColName(NewDealProposal{})
	newDealProposalEpochField = extractEpochFieldName(NewDealProposal{})
)

type NewDealProposalV8 struct {
	DealID          abi.DealID `bson:"_id"`
	Epoch           abi.ChainEpoch
	ProviderID      addr.Address
	ClientID        addr.Address
	MDealProposalV8 `bson:",inline"`
}

type NewDealProposal struct {
	DealID              abi.DealID `bson:"_id"`
	Epoch               abi.ChainEpoch
	ProviderID          addr.Address
	ClientID            addr.Address
	market.DealProposal `bson:",inline"`
}

func NewNewDealProposal(dealID abi.DealID, epoch abi.ChainEpoch, providerID, clientID addr.Address, dealProposal market.DealProposal) (*NewDealProposal, error) {
	return &NewDealProposal{
		DealID:       dealID,
		Epoch:        epoch,
		ProviderID:   providerID,
		ClientID:     clientID,
		DealProposal: dealProposal,
	}, nil
}

func NewNewDealProposalV8(dealID abi.DealID, epoch abi.ChainEpoch, providerID, clientID addr.Address, dealProposal market.DealProposal) (*NewDealProposalV8, error) {
	labelBytes := new(bytes.Buffer)
	err := dealProposal.Label.MarshalCBOR(labelBytes)
	if err != nil {
		return nil, fmt.Errorf("marshal label failed: %v", err)
	}

	newDealProposalV8 := &NewDealProposalV8{DealID: dealID, Epoch: epoch, ProviderID: providerID, ClientID: clientID, MDealProposalV8: MDealProposalV8{
		PieceCID:             dealProposal.PieceCID,
		PieceSize:            dealProposal.PieceSize,
		VerifiedDeal:         dealProposal.VerifiedDeal,
		Client:               dealProposal.Client,
		Provider:             dealProposal.Provider,
		Label:                labelBytes.Bytes(),
		StartEpoch:           dealProposal.StartEpoch,
		EndEpoch:             dealProposal.EndEpoch,
		StoragePricePerEpoch: dealProposal.StoragePricePerEpoch,
		ProviderCollateral:   dealProposal.ProviderCollateral,
		ClientCollateral:     dealProposal.ClientCollateral,
	}}

	return newDealProposalV8, nil
}

// CollectionName impl CollectionName
func (d *NewDealProposal) CollectionName() string {
	return newDealProposalColName
}

// EpochField impl common.Document
func (d *NewDealProposal) EpochField() *string {
	return &newDealProposalEpochField
}

// ResetPolicy impl common.Document
func (d *NewDealProposal) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(newDealProposalEpochField, lower, upper), true
}

// Indexes impl common.Indexed
func (d *NewDealProposal) Indexes() [][]string {
	return [][]string{
		[]string{newDealProposalEpochField, "VerifiedDeal"},
		[]string{"ProviderID", "-_id"},
		[]string{"ClientID", "-_id"},
		[]string{newDealProposalEpochField, "_id"},
	}
}

func (d *NewDealProposal) IsMutable() bool {
	return false
}

// CollectionName impl CollectionName
func (d *NewDealProposalV8) CollectionName() string {
	return newDealProposalColName
}

// EpochField impl common.Document
func (d *NewDealProposalV8) EpochField() *string {
	return &newDealProposalEpochField
}

// ResetPolicy impl common.Document
func (d *NewDealProposalV8) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(newDealProposalEpochField, lower, upper), true
}

// Indexes impl common.Indexed
func (d *NewDealProposalV8) Indexes() [][]string {
	return [][]string{
		[]string{newDealProposalEpochField, "VerifiedDeal"},
		[]string{"ProviderID", "-_id"},
		[]string{"ClientID", "-_id"},
	}
}

func (d *NewDealProposalV8) IsMutable() bool {
	return false
}
