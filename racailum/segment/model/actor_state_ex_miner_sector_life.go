package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*MinerSectorHealth)(nil)

	minerSectorHealthDetail     = getColName(MinerSectorHealth{})
	minerSectorHealthEpochField = extractEpochFieldName(MinerSectorHealth{})
)

// MinerSectorHealthDetail contains sector's health
type MinerSectorHealthDetail struct {
	Faults     uint64
	Recoveries uint64
	Unproven   uint64
	Active     uint64
	Live       uint64
	All        uint64

	ActiveSectorsQAPower abi.StoragePower
	FaultsQAPower        abi.StoragePower
	RecoveriesQAPower    abi.StoragePower
	UnprovenQAPower      abi.StoragePower

	ActiveSectorsRawPower abi.StoragePower
	FaultsRawPower        abi.StoragePower
	RecoveriesRawPower    abi.StoragePower
	UnprovenRawPower      abi.StoragePower

	TerminatedSectors uint64
}

// MinerSectorHealth shows sector health details for miner
type MinerSectorHealth struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MinerSectorHealthDetail
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
