package model

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                              common.IndexedDocument = (*MinerNewSectorNumber)(nil)
	minerNewSectorNumberEpochField                        = extractEpochFieldName(MinerNewSectorNumber{})
	minerNewSectorNumberColName                           = getColName(MinerNewSectorNumber{})
)

type MinerNewSectorNumber struct {
	ID           string `bson:"_id"`
	Miner        address.Address
	SectorNumber abi.SectorNumber
	Epoch        abi.ChainEpoch
}

func NewMinerNewSectorNumber(miner address.Address, sectorNumber abi.SectorNumber, epoch abi.ChainEpoch) *MinerNewSectorNumber {
	return &MinerNewSectorNumber{
		ID:           fmt.Sprintf("%v-%v", miner, sectorNumber),
		Miner:        miner,
		SectorNumber: sectorNumber,
		Epoch:        epoch,
	}
}

// CollectionName impl CollectionName
func (m *MinerNewSectorNumber) CollectionName() string {
	return minerNewSectorNumberColName
}

// EpochField impl common.Document
func (m *MinerNewSectorNumber) EpochField() *string {
	return &minerNewSectorNumberEpochField
}

// ResetPolicy impl common.Document
func (m *MinerNewSectorNumber) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerNewSectorNumberEpochField, lower, upper), true
}

func (m *MinerNewSectorNumber) Indexes() [][]string {
	return [][]string{
		[]string{"Miner", minerNewSectorNumberEpochField, "SectorNumber"},
		[]string{minerNewSectorNumberEpochField},
	}
}

func (m *MinerNewSectorNumber) IsMutable() bool {
	return false
}
