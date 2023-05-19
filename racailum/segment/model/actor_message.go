package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                      common.IndexedDocument = (*ActorMessage)(nil)
	actorMessageEpochField                        = extractEpochFieldName(ActorMessage{})
	actorMessageColName                           = getColName(ActorMessage{})
)

// ActorMessage records messages for actor
type ActorMessage struct {
	//ID         string `mir:"-" bson:"_id"`
	ActorID    address.Address
	Epoch      abi.ChainEpoch `mir:"-"`
	Cid        cid.Cid
	SignedCid  cid.Cid
	Value      abi.TokenAmount // int64
	MethodName string
	ExitCode   exitcode.ExitCode
	Type       string // "from" or "to"
	From       address.Address
	To         address.Address
	IsBlock    bool // 是否是块消息
}

func NewActorMessage(actorID address.Address, epoch abi.ChainEpoch, cid, signedCid cid.Cid, value abi.TokenAmount, methodName string, exitcode exitcode.ExitCode, mtype string, from, to address.Address, isBlock bool) (*ActorMessage, error) {
	return &ActorMessage{
		ActorID:    actorID,
		Epoch:      epoch,
		Cid:        cid,
		SignedCid:  signedCid,
		Value:      value,
		MethodName: methodName,
		ExitCode:   exitcode,
		Type:       mtype,
		From:       from,
		To:         to,
		IsBlock:    isBlock,
	}, nil
}

// Indexes impl common.Indexed
func (am *ActorMessage) Indexes() [][]string {
	return [][]string{
		[]string{"ActorID", "IsBlock", actorMessageEpochField},
		[]string{"ActorID", "IsBlock", "MethodName", actorMessageEpochField},
		[]string{"ActorID", "ExitCode", "Type", actorMessageEpochField, "Value"},
	}
}

// CollectionName impl common.Document
func (am *ActorMessage) CollectionName() string {
	return actorMessageColName
}

// EpochField impl common.Document
func (am *ActorMessage) EpochField() *string {
	return &actorMessageEpochField
}

// ResetPolicy impl common.Document
func (am *ActorMessage) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(actorMessageEpochField, lower, upper), true
}
