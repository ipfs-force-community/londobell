package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

var (
	_ common.Document = (*ActorBalance)(nil)

	actorBalanceColName    = getColName(ActorBalance{})
	actorBalanceEpochField = extractEpochFieldName(ActorBalance{})
)

func init() {
	schema.Register(
		schema.Model{
			Name: "actor-balance",
			D:    &ActorBalance{},
		},
	)
}

// EmptyActorBalance returns an empty detail with all fields initialized
func EmptyActorBalance() ActorBalance {
	return ActorBalance{
		Balance: abi.NewTokenAmount(0),
	}
}

// ActorBalance is the data model of ActorBalance
type ActorBalance struct {
	ActorStateExBasic `bson:",inline"`

	Addresses []address.Address
	Balance   abi.TokenAmount
}

// CollectionName impl CollectionName
func (a *ActorBalance) CollectionName() string {
	return actorBalanceColName
}

// EpochField impl common.Document
func (a *ActorBalance) EpochField() *string {
	return &actorBalanceEpochField
}

// ResetPolicy impl common.Document
func (a *ActorBalance) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(actorBalanceEpochField, lower, upper), true
}

func (a *ActorBalance) Indexes() [][]string {
	return [][]string{
		[]string{"Addresses"},
	}
}
