package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                     common.IndexedDocument = (*SectorClaim)(nil)
	sectorClaimEpochField                        = extractEpochFieldName(SectorClaim{})
	sectorClaimColName                           = getColName(SectorClaim{})
)

type SectorClaim struct {
	ClaimID   uint64 `bson:"_id"` // sorted
	Provider  abi.ActorID
	Client    abi.ActorID
	Data      cid.Cid
	Size      abi.PaddedPieceSize
	TermMin   abi.ChainEpoch
	TermMax   abi.ChainEpoch
	TermStart abi.ChainEpoch
	Sector    abi.SectorNumber
	//Drop         bool // drop只是当前指定sector不要claim(生命周期不允许)，后面还可以再加上(ExtendClaimTerms后)
	Epoch abi.ChainEpoch
}

func NewSectorClaim(claimID uint64, provider abi.ActorID, client abi.ActorID, data cid.Cid, size abi.PaddedPieceSize, termMin abi.ChainEpoch, termMax abi.ChainEpoch, termStart abi.ChainEpoch, sector abi.SectorNumber, epoch abi.ChainEpoch) *SectorClaim {
	return &SectorClaim{
		ClaimID:   claimID,
		Provider:  provider,
		Client:    client,
		Data:      data,
		Size:      size,
		TermMin:   termMin,
		TermMax:   termMax,
		TermStart: termStart,
		Sector:    sector,
		Epoch:     epoch,
	}
}

// CollectionName impl CollectionName
func (s *SectorClaim) CollectionName() string {
	return sectorClaimColName
}

// EpochField impl common.Document
func (s *SectorClaim) EpochField() *string {
	return &sectorClaimEpochField
}

// ResetPolicy impl common.Document
func (s *SectorClaim) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(sectorClaimEpochField, lower, upper), true
}

func (s *SectorClaim) Indexes() [][]string {
	return [][]string{
		[]string{"Provider"},
		[]string{"Provider", "Sector"},
		[]string{"Provider", "Drop"},
	}
}

func (s *SectorClaim) IsMutable() bool {
	return true
}
