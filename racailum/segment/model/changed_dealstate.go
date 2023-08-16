package model

import (
	"fmt"

	"github.com/filecoin-project/lotus/chain/actors/builtin/market"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                          common.IndexedDocument = (*ChangedDealState)(nil)
	changedDealStateEpochField                        = extractEpochFieldName(ChangedDealState{})
	changedDealStateColName                           = getColName(ChangedDealState{})
)

type ChangedDealState struct {
	ID               string `bson:"_id"`
	DealID           abi.DealID
	Epoch            abi.ChainEpoch
	market.DealState `bson:",inline"`
	Added            bool // new created
	Removed          bool
}

func NewChangedDealState(dealID abi.DealID, dealState market.DealState, epoch abi.ChainEpoch, added, removed bool) *ChangedDealState {
	return &ChangedDealState{
		ID:        fmt.Sprintf("%v-%v", dealID, epoch),
		DealID:    dealID,
		Epoch:     epoch,
		DealState: dealState,
		Added:     added,
		Removed:   removed,
	}
}

// CollectionName impl CollectionName
func (m *ChangedDealState) CollectionName() string {
	return changedDealStateColName
}

// EpochField impl common.Document
func (m *ChangedDealState) EpochField() *string {
	return &changedDealStateEpochField
}

// ResetPolicy impl common.Document
func (m *ChangedDealState) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(changedDealStateEpochField, lower, upper), true
}

func (m *ChangedDealState) Indexes() [][]string {
	return [][]string{
		[]string{changedDealStateEpochField, "DealID", "Added"},
		[]string{changedDealStateEpochField, "DealID", "Removed"},
		[]string{"DealID"},
	}
}

func (m *ChangedDealState) IsMutable() bool {
	return false
}
