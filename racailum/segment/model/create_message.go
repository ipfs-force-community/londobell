package model

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v11/eam"
	"github.com/filecoin-project/go-state-types/builtin/v11/power"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"
	"golang.org/x/exp/slices"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	ConstructorMethod                              = "Constructor"
	CreateExternal                                 = "CreateExternal"
	CreateMiner                                    = "CreateMiner"
	CreateMethods                                  = []string{CreateMiner, CreateExternal, "Exec", ConstructorMethod}
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
	From       address.Address
	To         address.Address
	IsBlock    bool            // 是否是块消息
	Caller     address.Address //constructor caller address
	ActorID    address.Address //CreateExternal created
}

func IsOkCreateMessage(methodName string, exitCode int64) bool {
	return slices.Contains(CreateMethods, methodName) && exitCode == 0
}

func NewCreateMessage(epoch abi.ChainEpoch, cid, signedCid cid.Cid, value abi.TokenAmount, methodName string, exitcode exitcode.ExitCode, from, to address.Address, isBlock bool, seq []int, callerMap map[string]address.Address, returnObj cbor.Er, raw *common.ExecutionTraceCompact) (*CreateMessage, error) {

	am := &CreateMessage{
		Epoch:      epoch,
		Cid:        cid,
		SignedCid:  signedCid,
		Value:      value,
		MethodName: methodName,
		From:       from,
		To:         to,
		IsBlock:    isBlock,
	}
	am.genID(epoch, seq)
	if methodName == ConstructorMethod {
		parts := strings.Split(am.ID, "-")

		// Take the first two segments
		if len(parts) >= 2 {
			callerID := parts[0] + "-" + parts[1]
			if caller, ok := callerMap[callerID]; ok {
				am.Caller = caller
			} else {
				return nil, fmt.Errorf("no caller in callerAddrMap")
			}
		} else {
			return nil, fmt.Errorf("get constructor caller err,id: %s", am.ID)
		}
	} else if methodName == CreateExternal || methodName == CreateMiner {
		if len(raw.MsgRct.Return) > 0 && returnObj != nil {
			if err := returnObj.UnmarshalCBOR(bytes.NewReader(raw.MsgRct.Return)); err != nil {
				return nil, fmt.Errorf("unmarshal return: %w", err)
			}

			addr, err := parse(methodName, returnObj)
			if err != nil {
				return nil, err
			}
			am.ActorID = addr
		}
	}

	return am, nil
}

func CompareStructPointers(a interface{}, b interface{}) bool {
	valueA := reflect.ValueOf(a)
	valueB := reflect.ValueOf(b)

	if valueA.Kind() != reflect.Ptr || valueB.Kind() != reflect.Ptr {
		return false
	}

	elemTypeA := valueA.Elem().Type()
	elemTypeB := valueB.Elem().Type()
	if elemTypeA.Kind() != reflect.Struct || elemTypeB.Kind() != reflect.Struct {
		return false
	}

	if elemTypeA.String() != elemTypeB.String() {
		return false
	}

	numFields := elemTypeA.NumField()

	for i := 0; i < numFields; i++ {
		fieldTypeA := elemTypeA.Field(i)
		fieldTypeB := elemTypeB.Field(i)

		if fieldTypeA.Name != fieldTypeB.Name || fieldTypeA.Type != fieldTypeB.Type {
			return false
		}

	}

	return true
}

func parse(methodName string, obj interface{}) (address.Address, error) {
	if methodName == CreateExternal {
		if ret, ok := obj.(*eam.CreateExternalReturn); ok {
			addr, err := address.NewIDAddress(ret.ActorID)
			if err != nil {
				return address.Address{}, fmt.Errorf("parse's convert addr err %w", err)
			}
			return addr, nil
		}
		// if builtin version changed use the following logic
		switch reflect.TypeOf(obj).String() {
		case "*eam.CreateExternalReturn":
			if CompareStructPointers(obj, &eam.CreateExternalReturn{}) {
				addr, err := address.NewIDAddress(obj.(*eam.CreateExternalReturn).ActorID)
				if err != nil {
					return address.Address{}, fmt.Errorf("parse's convert addr err %w", err)
				}
				return addr, nil
			}

			return address.Address{}, fmt.Errorf("parse CreateExternal err type: %s", reflect.TypeOf(obj))

		default:
			return address.Address{}, fmt.Errorf("parse CreateExternal err type: %s", reflect.TypeOf(obj))
		}
	} else if methodName == CreateMiner {
		if ret, ok := obj.(*power.CreateMinerReturn); ok {

			return ret.IDAddress, nil
		}
		// if builtin version changed use the following logic
		switch reflect.TypeOf(obj).String() {
		case "*power.CreateMinerReturn":
			if CompareStructPointers(obj, &power.CreateMinerReturn{}) {
				return obj.(*power.CreateMinerReturn).IDAddress, nil
			}

			return address.Address{}, fmt.Errorf("parse CreateMiner err type: %s", reflect.TypeOf(obj))

		default:
			return address.Address{}, fmt.Errorf("CreateMiner err type: %s", reflect.TypeOf(obj))
		}
	}
	return address.Address{}, fmt.Errorf("CreateMiner err type: %s", reflect.TypeOf(obj))
}

// Indexes impl common.Indexed
func (am *CreateMessage) Indexes() [][]string {
	return [][]string{
		{"IsBlock", createMessageEpochField},
		{"Method"},
		{"Cid", createMessageEpochField},
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
