package model

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	ConstructorMethod                              = "Constructor"
	CreateMethods                                  = []string{"CreateMiner", "CreateExternal", "Exec", ConstructorMethod}
	_                       common.IndexedDocument = (*CreateMessage)(nil)
	createMessageEpochField                        = extractEpochFieldName(CreateMessage{})
	createMessageColName                           = getColName(CreateMessage{})
)

// CreateMessage records messages for create
type CreateMessage struct {
	ID         string         `mir:"-" bson:"_id"`
	Epoch      abi.ChainEpoch `mir:"-"`
	Cid        cid.Cid
	SignedCid  cid.Cid
	Value      abi.TokenAmount // int64
	MethodName string
	ExitCode   exitcode.ExitCode
	From       address.Address
	To         address.Address
	IsBlock    bool            // 是否是块消息
	Return     []byte          //msg return
	Caller     address.Address //constructor caller address
}

func NewCreateMessage(epoch abi.ChainEpoch, cid, signedCid cid.Cid, value abi.TokenAmount, methodName string, exitcode exitcode.ExitCode, from, to address.Address, isBlock bool, seq []int, msgRet []byte) (*CreateMessage, error) {

	am := &CreateMessage{
		Epoch:      epoch,
		Cid:        cid,
		SignedCid:  signedCid,
		Value:      value,
		MethodName: methodName,
		ExitCode:   exitcode,
		From:       from,
		To:         to,
		IsBlock:    isBlock,
		Return:     msgRet,
	}

	am.genID(epoch, seq)

	return am, nil
}

// Indexes impl common.Indexed
func (am *CreateMessage) Indexes() [][]string {
	return [][]string{
		{"ActorID", "IsBlock", createMessageEpochField},
		{"ActorID", "IsBlock", "MethodName", createMessageEpochField},
		{"ActorID", "ExitCode", "Type", createMessageEpochField, "Value"},
	}
}

// CollectionName impl common.Document
func (am *CreateMessage) CollectionName() string {
	return createMessageColName
}

// EpochField impl common.Document
func (am *CreateMessage) EpochField() *string {
	return &createMessageEpochField
}

// ResetPolicy impl common.Document
func (am *CreateMessage) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(createMessageEpochField, lower, upper), true
}

func (am *CreateMessage) genID(epoch abi.ChainEpoch, seq []int) {
	seqStrs := make([]string, 0, len(seq))
	for i := range seq {
		seqStrs = append(seqStrs, fmt.Sprintf("%05d", seq[i]))
	}

	am.ID = fmt.Sprintf("%d-%s", epoch, strings.Join(seqStrs, "-"))
}

func (am *CreateMessage) IsMutable() bool {
	return false
}
