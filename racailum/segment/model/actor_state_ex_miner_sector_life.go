package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/dtynn/londobell/common"
)

var (
	_ common.IndexedDocument = (*MinerFunds)(nil)

	minerSectorHealthDetail     = getColName(MinerSectorHealth{})
	minerSectorHealthEpochField = extractEpochFieldName(MinerSectorHealth{})
)

// MinerSectorHealthDetail contains sector's health
type MinerSectorHealthDetail struct {
	TotalSectors uint64
	LiveSectors  uint64
	Faults       uint64
	Recoveries   uint64
}

// MinerSectorHealth shows sector health details for miner
type MinerSectorHealth struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MinerFundsDetail
}

// CollectionName impl common.Document
func (m *MinerSectorHealth) CollectionName() string {
	return minerSectorHealthDetail
}

// EpochField impl common.Document
func (m *MinerSectorHealth) EpochField() *string {
	return &minerSectorHealthEpochField
}

// ResetPolicy impl common.Document
func (m *MinerSectorHealth) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerSectorHealthEpochField, lower, upper), true
}
