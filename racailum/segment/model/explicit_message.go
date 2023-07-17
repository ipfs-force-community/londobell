package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                         common.IndexedDocument = (*ExplicitMessage)(nil)
	explicitMessageEpochField                        = extractEpochFieldName(ExplicitMessage{})
	explicitMessageColName                           = getColName(ExplicitMessage{})
)

// ExplicitMessage records explicit messages
type ExplicitMessage struct {
	Cid        cid.Cid         `mir:"-" bson:"_id"`
	Epoch      abi.ChainEpoch  `mir:"-"`
	Value      abi.TokenAmount // int64
	MethodName string
	ExitCode   exitcode.ExitCode
	From       address.Address
	To         address.Address
}

func NewExplicitMessage(cid cid.Cid, epoch abi.ChainEpoch, value abi.TokenAmount, methodName string, exitcode exitcode.ExitCode, from, to address.Address) *ExplicitMessage {
	return &ExplicitMessage{
		Cid:        cid,
		Epoch:      epoch,
		Value:      value,
		MethodName: methodName,
		ExitCode:   exitcode,
		From:       from,
		To:         to,
	}
}

// Indexes impl common.Indexed
func (em *ExplicitMessage) Indexes() [][]string {
	return [][]string{
		[]string{actorMessageEpochField},
		[]string{"MethodName", actorMessageEpochField},
		[]string{"ExitCode", actorMessageEpochField, "Value"},
	}
}

// CollectionName impl common.Document
func (em *ExplicitMessage) CollectionName() string {
	return explicitMessageColName
}

// EpochField impl common.Document
func (em *ExplicitMessage) EpochField() *string {
	return &explicitMessageEpochField
}

// ResetPolicy impl common.Document
func (em *ExplicitMessage) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(explicitMessageEpochField, lower, upper), true
}

func (em *ExplicitMessage) IsMutable() bool {
	return false
}
