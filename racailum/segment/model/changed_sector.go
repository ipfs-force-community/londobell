package model

import (
	"fmt"

	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                       common.IndexedDocument = (*ChangedSector)(nil)
	changedSectorEpochField                        = extractEpochFieldName(ChangedSector{})
	changedSectorColName                           = getColName(ChangedSector{})
)

type ChangedSector struct {
	ID                      string `bson:"_id"`
	Miner                   address.Address
	Epoch                   abi.ChainEpoch
	miner.SectorOnChainInfo `bson:",inline"`
	Added                   bool // new created
	Removed                 bool
}

func NewChangedSector(sectorOnChainInfo miner.SectorOnChainInfo, miner address.Address, epoch abi.ChainEpoch, added, removed bool) *ChangedSector {
	return &ChangedSector{
		ID:                fmt.Sprintf("%v-%v-%v", miner, sectorOnChainInfo.SectorNumber, epoch),
		Miner:             miner,
		Epoch:             epoch,
		SectorOnChainInfo: sectorOnChainInfo,
		Added:             added,
		Removed:           removed,
	}
}

// CollectionName impl CollectionName
func (m *ChangedSector) CollectionName() string {
	return changedSectorColName
}

// EpochField impl common.Document
func (m *ChangedSector) EpochField() *string {
	return &changedSectorEpochField
}

// ResetPolicy impl common.Document
func (m *ChangedSector) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(changedSectorEpochField, lower, upper), true
}

func (m *ChangedSector) Indexes() [][]string {
	return [][]string{
		[]string{"Miner", minerSectorEpochField, "SectorNumber", "Removed"},
		[]string{"Miner", minerSectorEpochField, "Added"},
		[]string{"Miner", minerSectorEpochField, "Expiration", "Removed"},
		[]string{"Miner", "SimpleQaPower"},
		[]string{"Miner", "DealIDs"},
	}
}

func (m *ChangedSector) IsMutable() bool {
	return false
}
