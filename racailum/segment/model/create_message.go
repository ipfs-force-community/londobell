package model

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v11/eam"
	vinit "github.com/filecoin-project/go-state-types/builtin/v11/init"
	"github.com/filecoin-project/go-state-types/builtin/v11/power"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"
	"golang.org/x/exp/slices"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

var (
	ConstructorMethod                              = "Constructor"
	CreateExternal                                 = "CreateExternal"
	CreateMiner                                    = "CreateMiner"
	Exec                                           = "Exec"
	CreateMethods                                  = []string{CreateMiner, CreateExternal, Exec, ConstructorMethod}
	_                       common.IndexedDocument = (*CreateMessage)(nil)
	createMessageEpochField                        = extractEpochFieldName(CreateMessage{})
	createMessageColName                           = getColName(CreateMessage{})
)

func init() {
	schema.Register(
		schema.Model{
			Name: "create-message",
			D:    &CreateMessage{},
		},
	)
}

// CreateMessage records messages for create
type CreateMessage struct {
	ID            string         `mir:"-" bson:"_id"`
	Epoch         abi.ChainEpoch `mir:"-"`
	Cid           cid.Cid
	SignedCid     cid.Cid
	Value         abi.TokenAmount // int64
	MethodName    string
	From          address.Address
	To            address.Address
	IsBlock       bool            // 是否是块消息
	Caller        address.Address //constructor caller address
	ActorID       address.Address //CreateExternal created
	RootCid       cid.Cid         `mir:"-"`
	RootSignedCid cid.Cid         `mir:"-"`
}

func IsOkCreateMessage(methodName string, exitCode int64) bool {
	return slices.Contains(CreateMethods, methodName) && exitCode == 0
}

func NewCreateMessage(ctx *extract.Ctx, epoch abi.ChainEpoch, cid, signedCid cid.Cid, value abi.TokenAmount, methodName string, exitcode exitcode.ExitCode, from, to address.Address, isBlock bool, seq []int, callerMap map[string]address.Address, returnObj cbor.Er, raw *common.ExecutionTraceCompact, IDCidMap map[string][2]cid.Cid) (*CreateMessage, error) {
	elog := ctx.L.With("NewCreateMessage", cid)
	cm := &CreateMessage{
		Epoch:      epoch,
		Cid:        cid,
		SignedCid:  signedCid,
		Value:      value,
		MethodName: methodName,
		From:       from,
		To:         to,
		IsBlock:    isBlock,
	}
	cm.genID(epoch, seq)
	err := cm.genRootids(IDCidMap)
	if err != nil {
		elog.Warn(err)
	}
	if methodName == ConstructorMethod {
		parts := strings.Split(cm.ID, "-")

		// Take the first two segments
		if len(parts) >= 2 {
			callerID := parts[0] + "-" + parts[1]
			if caller, ok := callerMap[callerID]; ok {
				cm.Caller = caller
			} else {
				return nil, fmt.Errorf("no caller in callerAddrMap")
			}
		} else {
			return nil, fmt.Errorf("get constructor caller err,id: %s", cm.ID)
		}
	} else if methodName == CreateExternal || methodName == CreateMiner || methodName == Exec {
		if len(raw.MsgRct.Return) > 0 && returnObj != nil {
			if err := returnObj.UnmarshalCBOR(bytes.NewReader(raw.MsgRct.Return)); err != nil {
				return nil, fmt.Errorf("unmarshal return: %w", err)
			}
			addr, err := parse(methodName, returnObj)
			if err != nil {
				return nil, err
			}
			cm.ActorID = addr
		}
	}

	return cm, nil
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
	} else if methodName == Exec {
		if ret, ok := obj.(*vinit.ExecReturn); ok {

			return ret.IDAddress, nil
		}
		// if builtin version changed use the following logic
		switch reflect.TypeOf(obj).String() {
		case "*init.ExecReturn":
			if CompareStructPointers(obj, &vinit.ExecReturn{}) {
				return obj.(*vinit.ExecReturn).IDAddress, nil
			}

			return address.Address{}, fmt.Errorf("parse Exec err type: %s", reflect.TypeOf(obj))

		default:
			return address.Address{}, fmt.Errorf("Exec err type: %s", reflect.TypeOf(obj))
		}
	}
	return address.Address{}, fmt.Errorf("parse err method: %s, type: %s", methodName, reflect.TypeOf(obj))
}

// Indexes impl common.Indexed
func (cm *CreateMessage) Indexes() [][]string {
	return [][]string{
		{"IsBlock", createMessageEpochField},
		{"Method"},
		{"Cid", createMessageEpochField},
		{"ActorID"},
		{"Caller"},
	}
}

// CollectionName impl common.Document
func (cm *CreateMessage) CollectionName() string {
	return createMessageColName
}

// EpochField impl common.Document
func (cm *CreateMessage) EpochField() *string {
	return &createMessageEpochField
}

// ResetPolicy impl common.Document
func (cm *CreateMessage) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(createMessageEpochField, lower, upper), true
}

func (cm *CreateMessage) genID(epoch abi.ChainEpoch, seq []int) {
	seqStrs := make([]string, 0, len(seq))
	for i := range seq {
		seqStrs = append(seqStrs, fmt.Sprintf("%05d", seq[i]))
	}

	cm.ID = fmt.Sprintf("%d-%s", epoch, strings.Join(seqStrs, "-"))
}

func (cm *CreateMessage) IsMutable() bool {
	return false
}

// get root Cid SignedCid
func (cm *CreateMessage) genRootids(m map[string][2]cid.Cid) error {
	if cm.IsBlock {
		return nil
	}
	rootID, err := GetRootID(cm.ID)
	if err != nil {
		return err
	}
	cm.RootCid = m[rootID][0]
	cm.RootSignedCid = m[rootID][1]
	return nil
}
