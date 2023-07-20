package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

var (
	_ common.Document = (*ActorAddress)(nil)

	actorAddressColName    = getColName(ActorAddress{})
	actorAddressEpochField = extractEpochFieldName(ActorAddress{})
)

func init() {
	schema.Register(
		schema.Model{
			Name: "actor-address",
			D:    &ActorAddress{},
		},
	)
}

// ActorAddress is the data model of ActorAddress
type ActorAddress struct {
	ActorID          address.Address `bson:"_id"`
	RobustAddress    address.Address
	DelegatedAddress address.Address
	Epoch            abi.ChainEpoch
}

// CollectionName impl CollectionName
func (a *ActorAddress) CollectionName() string {
	return actorAddressColName
}

// EpochField impl common.Document
func (a *ActorAddress) EpochField() *string {
	return &actorAddressEpochField
}

// ResetPolicy impl common.Document
func (a *ActorAddress) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(actorAddressEpochField, lower, upper), true
}

func (a *ActorAddress) Indexes() [][]string {
	return [][]string{
		[]string{"RobustAddress"},
		[]string{"DelegatedAddress"},
		[]string{actorAddressEpochField},
	}
}

func (a *ActorAddress) IsMutable() bool {
	return false
}
