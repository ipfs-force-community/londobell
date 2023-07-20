package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

var (
	_ common.Document = (*ChangedActor)(nil)

	changedActorColName    = getColName(ChangedActor{})
	changedActorEpochField = extractEpochFieldName(ChangedActor{})
)

func init() {
	schema.Register(
		schema.Model{
			Name: "changed-actor",
			D:    &ChangedActor{},
		},
	)
}

// ChangedActor is the data model of ChangedActor
type ChangedActor struct {
	ID      cid.Cid `bson:"_id"`
	Epoch   abi.ChainEpoch
	ActorID address.Address
	Balance abi.TokenAmount
	Code    string
	Address *address.Address // Deterministic Address.
}

// CollectionName impl CollectionName
func (a *ChangedActor) CollectionName() string {
	return changedActorColName
}

// EpochField impl common.Document
func (a *ChangedActor) EpochField() *string {
	return &changedActorEpochField
}

// ResetPolicy impl common.Document
func (a *ChangedActor) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(changedActorEpochField, lower, upper), true
}

func (a *ChangedActor) Indexes() [][]string {
	return [][]string{
		[]string{"ActorID"},
		[]string{"Code"},
		[]string{changedActorEpochField},
		[]string{"ActorID", changedActorEpochField},
	}
}

func (a *ChangedActor) IsMutable() bool {
	return false
}
