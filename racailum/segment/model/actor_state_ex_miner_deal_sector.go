package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*MinerDealSector)(nil)

	minerDealSectorColName    = getColName(MinerDealSector{})
	minerDealSectorEpochField = extractEpochFieldName(MinerDealSector{})
)

// MinerDealSector contains several token amounts
type MinerDealSector struct {
	ID                 string `bson:"_id"`
	Epoch              abi.ChainEpoch
	SectorNumber       abi.SectorNumber
	SealProof          abi.RegisteredSealProof
	DealIDs            []abi.DealID
	DealWeight         abi.DealWeight  // Integral of active deals over sector lifetime
	VerifiedDealWeight abi.DealWeight  // Integral of active verified deals over sector lifetime
	InitialPledge      abi.TokenAmount // Pledge collected to commit this sector
	QAPower            abi.StoragePower
	Miner              address.Address
}

// CollectionName impl common.Document
func (m *MinerDealSector) CollectionName() string {
	return minerDealSectorColName
}

// EpochField impl common.Document
func (m *MinerDealSector) EpochField() *string {
	return &minerDealSectorEpochField
}

// ResetPolicy impl common.Document
func (m *MinerDealSector) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerDealSectorEpochField, lower, upper), true
}

func (m *MinerDealSector) Indexes() [][]string {
	return [][]string{
		[]string{minerDealSectorEpochField, "Miner"},
	}
}
