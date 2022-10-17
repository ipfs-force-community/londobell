package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs/go-cid"
)

var (
	_                     common.IndexedDocument = (*Allocations)(nil)
	allocationsColName                           = getColName(Allocations{})
	allocationsEpochField                        = extractEpochFieldName(Allocations{})
)

type AllocationsDetail struct {
	AllocationID uint64
	Client       abi.ActorID
	Provider     abi.ActorID
	Data         cid.Cid
	Size         abi.PaddedPieceSize
	TermMin      abi.ChainEpoch
	TermMax      abi.ChainEpoch
	Expiration   abi.ChainEpoch
}

type Allocations struct {
	ActorStateExBasic `bson:",inline"`
	Detail            AllocationsDetail
}

func (a *Allocations) CollectionName() string {
	return allocationsColName
}

func (a *Allocations) EpochField() *string {
	return &allocationsEpochField
}

func (a *Allocations) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(allocationsEpochField, lower, upper), true
}

func (a *Allocations) Indexes() [][]string {
	return [][]string{
		[]string{"Addr"},
		[]string{allocationsEpochField, "Addr"},
		[]string{allocationsEpochField, "Addr", "Detail.AllocationID"},
	}
}
