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

type Claims struct {
	ID        string `bson:"_id"`
	Epoch     abi.ChainEpoch
	ClaimID   uint64
	Provider  abi.ActorID // todo: ActorID is Provider, Is there going to be an illegal situation？
	Client    abi.ActorID
	Data      cid.Cid
	Size      abi.PaddedPieceSize
	TermMin   abi.ChainEpoch
	TermMax   abi.ChainEpoch
	TermStart abi.ChainEpoch
	Sector    abi.SectorNumber
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
		[]string{"Provider"},
		[]string{"Client"},
		[]string{"Data"},
		[]string{claimsEpochField, "Provider"},
		[]string{claimsEpochField, "Client"},
		[]string{claimsEpochField, "Provider", "ClaimID"},
		[]string{claimsEpochField, "Client", "ClaimID"},
	}
}
