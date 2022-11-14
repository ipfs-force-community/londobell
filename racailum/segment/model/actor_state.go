package model

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"
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
		ID:       fmt.Sprintf("%v-%v", head, head.Addr),
		Head:     head.Head,
		Addr:     head.Addr,
		CodeName: builtin.ActorNameByCode(head.Code),
		Code:     head.Code,
		Balance:  head.Balance,
		Epoch:    head.Epoch,
		Detail:   raw,
	}, nil
}

// ActorState is the schema of actor states
type ActorState struct {
	ID       string  `mir:"-" bson:"_id"`
	Head     cid.Cid `mir:"-"`
	Addr     address.Address
	CodeName string
	Code     cid.Cid
	Balance  types.BigInt
	Epoch    abi.ChainEpoch
	Detail   ActorStateDetail
}

// Indexes impl common.Indexed
func (a *ActorState) Indexes() [][]string {
	return [][]string{
		[]string{actorStateEpochField, "Code", "Addr"},
		[]string{actorStateEpochField, "CodeName", "Addr"},
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
