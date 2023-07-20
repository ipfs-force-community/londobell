package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	verifreg9 "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                     common.IndexedDocument = (*Allocations)(nil)
	allocationsColName                           = getColName(Allocations{})
	allocationsEpochField                        = extractEpochFieldName(Allocations{})
)

type AllocationID = verifreg9.AllocationId

type Allocations struct {
	ID           string `bson:"_id"`
	Epoch        abi.ChainEpoch
	AllocationID AllocationID
	Client       abi.ActorID // todo: ActorID is Client, Is there going to be an illegal situation？
	Provider     abi.ActorID
	Data         cid.Cid
	Size         abi.PaddedPieceSize
	TermMin      abi.ChainEpoch
	TermMax      abi.ChainEpoch
	Expiration   abi.ChainEpoch
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
		[]string{"Client"},
		[]string{"Provider"},
		[]string{"Data"},
		[]string{allocationsEpochField, "Client"},
		[]string{allocationsEpochField, "Provider"},
		[]string{allocationsEpochField, "Client", "AllocationID"},
		[]string{allocationsEpochField, "Provider", "AllocationID"},
	}
}

func (a *Allocations) IsMutable() bool {
	return false
}
