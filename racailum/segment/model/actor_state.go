package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mgoutil/mcodec"
)

func init() {
	mcodec.RegisterSchemaType(new(ActorStateDetail))
}

var _ common.IndexedDocument = (*ActorState)(nil)

var (
	actorStateColName    = getColName(ActorState{})
	actorStateEpochField = extractEpochFieldName(ActorState{})
)

// ActorStateDetail is a type alias
type ActorStateDetail cbor.Er

// NewActorState converts raw to ActorStateFromRaw
func NewActorState(head *common.ActorHead, raw cbor.Er) (*ActorState, error) {
	return &ActorState{
		Head:    head.Head,
		Addr:    head.Addr,
		Code:    builtin.ActorNameByCode(head.Code),
		Balance: head.Balance,
		Epoch:   head.Epoch,
		Detail:  raw,
	}, nil
}

// ActorState is the schema of actor states
type ActorState struct {
	Head    cid.Cid `mir:"-" bson:"_id"`
	Addr    address.Address
	Code    string
	Balance types.BigInt
	Epoch   abi.ChainEpoch
	Detail  ActorStateDetail
}

// Indexes impl common.Indexed
func (a *ActorState) Indexes() [][]string {
	return [][]string{
		[]string{actorStateEpochField, "Code", "Addr"},
	}
}

// CollectionName impl common.Document
func (a *ActorState) CollectionName() string {
	return actorStateColName
}

// EpochField impl common.Document
func (a *ActorState) EpochField() *string {
	return &actorStateEpochField
}

// ResetPolicy impl common.Document
func (a *ActorState) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(actorStateEpochField, lower, upper), true
}
