package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs/go-cid"
)

var (
	_                common.IndexedDocument = (*Claims)(nil)
	claimsColName                           = getColName(Claims{})
	claimsEpochField                        = extractEpochFieldName(Claims{})
)

type ClaimsDetail struct {
	ClaimID   uint64
	Provider  abi.ActorID
	Client    abi.ActorID
	Data      cid.Cid
	Size      abi.PaddedPieceSize
	TermMin   abi.ChainEpoch
	TermMax   abi.ChainEpoch
	TermStart abi.ChainEpoch
	Sector    abi.SectorNumber
}

type Claims struct {
	ActorStateExBasic `bson:",inline"`
	Detail            ClaimsDetail
}

func (c *Claims) CollectionName() string {
	return claimsColName
}

func (c *Claims) EpochField() *string {
	return &claimsEpochField
}

func (c *Claims) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(claimsEpochField, lower, upper), true
}

func (c *Claims) Indexes() [][]string {
	return [][]string{
		[]string{"Addr"},
		[]string{claimsEpochField, "Addr"},
		[]string{claimsEpochField, "Addr", "Detail.ClaimID"},
	}
}
