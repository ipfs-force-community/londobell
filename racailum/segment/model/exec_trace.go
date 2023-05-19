package model

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"
	"github.com/ipfs-force-community/londobell/lib/mir"
)

func init() {
	mcodec.RegisterSchemaType(new(ExecTraceReturn))
}

// NSGasTraceNames is the name of the dict GasTraceNames
const NSGasTraceNames = "GasTraceNames"

var (
	execTraceColName    = getColName(ExecTrace{})
	execTraceEpochField = extractEpochFieldName(ExecTrace{})
	execGasColName      = getColName(ExecGas{})
	execGasEpochField   = extractEpochFieldName(ExecGas{})
)

var (
	_ common.IndexedDocument = (*ExecTrace)(nil)
	_ common.Document        = (*ExecGas)(nil)
)

// NewExecTrace converts raw exec trace struct to ExecTrace*
func NewExecTrace(
	ctx context.Context,
	dal common.DAL,
	mcid cid.Cid,
	signedCid cid.Cid,
	epoch abi.ChainEpoch,
	seq []int,
	raw *common.ExecutionTraceCompact,
	returnObj cbor.Er,
	cost *api.MsgGasCost, meth string,
) (*ExecTrace, *ExecGas, error) {
	me := &ExecTrace{
		Cid:          mcid,
		SignedCid:    signedCid,
		Epoch:        epoch,
		Seq:          seq,
		Depth:        len(seq),
		Ver:          build.CurrentCommit,
		SubCallCount: len(raw.Subcalls),
		GasCost:      cost,
	}

	if err := mir.Mirror(me, raw); err != nil {
		return nil, nil, fmt.Errorf("mirroring message exec: %w", err)
	}

	me.Msg.MethodName = meth
	me.SeqIndex = make([][]int, len(seq))
	for i := range seq {
		me.SeqIndex[i] = seq[:i+1]
	}

	if len(raw.MsgRct.Return) > 0 && returnObj != nil {
		if err := returnObj.UnmarshalCBOR(bytes.NewReader(raw.MsgRct.Return)); err != nil {
			return nil, nil, fmt.Errorf("unmarshal return: %w", err)
		}

		me.Detail.Return = returnObj
	}

	me.genID()

	//var mg *ExecGas
	//
	//if len(raw.GasCharges) > 0 {
	//	mg = &ExecGas{
	//		Epoch:   epoch,
	//		Charges: make([]common.GasTraceCompact, len(raw.GasCharges)),
	//	}
	//
	//	for i := range raw.GasCharges {
	//		charge := raw.GasCharges[i]
	//
	//		nameIdx, err := dal.LookupEnum(ctx, NSGasTraceNames, charge.Name)
	//		if err != nil {
	//			return nil, nil, fmt.Errorf("lookup for gas-trace-name index for %s in dict: %w", charge.Name, err)
	//		}
	//
	//		charge.Name = fmt.Sprintf("$%d", nameIdx)
	//		mg.Charges[i] = charge
	//	}
	//
	//	mg.ID = me.ID
	//}

	return me, nil, nil
}

// ExecTraceReturn is a type alias
type ExecTraceReturn cbor.Er

// ExecTrace is the schema of *api.ExecutionTrace
type ExecTrace struct {
	ID string `mir:"-" bson:"_id"`

	Cid       cid.Cid        `mir:"-"`
	SignedCid cid.Cid        `mir:"-" bson:"SignedCid,omitempty"`
	Epoch     abi.ChainEpoch `mir:"-"`
	Seq       []int          `mir:"-"`
	Depth     int            `mir:"-"`

	Ver string `mir:"-"`

	Msg struct {
		From       address.Address
		To         address.Address
		Method     abi.MethodNum
		Value      abi.TokenAmount
		MethodName string
	}

	// raw infos
	MsgRct types.MessageReceipt
	Error  string

	SeqIndex [][]int `mir:"-"`

	SubCallCount int `mir:"-"`

	Detail struct {
		Return ExecTraceReturn
	} `mir:"-"`

	GasCost *api.MsgGasCost `mir:"-"`
}

// Indexes impl common.Indexed
func (et *ExecTrace) Indexes() [][]string {
	return [][]string{
		[]string{execTraceEpochField, "Msg.To", "Msg.Method", "MsgRct.ExitCode"},
		[]string{execTraceEpochField, "Msg.To", "Seq"},
		[]string{"Cid"},
		[]string{"SignedCid"},

		[]string{"Depth", execTraceEpochField},
		[]string{"Depth", "Msg.MethodName", execTraceEpochField},
	}
}

// CollectionName impls common.Document
func (et *ExecTrace) CollectionName() string {
	return execTraceColName
}

// EpochField impl common.Document
func (et *ExecTrace) EpochField() *string {
	return &execTraceEpochField
}

// ResetPolicy impls common.Document
func (et *ExecTrace) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(execTraceEpochField, lower, upper), true
}

func (et *ExecTrace) genID() {
	seqStrs := make([]string, 0, len(et.Seq))
	for i := range et.Seq {
		seqStrs = append(seqStrs, fmt.Sprintf("%05d", et.Seq[i]))
	}

	et.ID = fmt.Sprintf("%d-%s", et.Epoch, strings.Join(seqStrs, "-"))
}

// ExecGas stores gas charges in another collection
type ExecGas struct {
	ID      string `mir:"-" bson:"_id"`
	Epoch   abi.ChainEpoch
	Charges []common.GasTraceCompact
}

// CollectionName impls common.Document
func (eg *ExecGas) CollectionName() string {
	return execGasColName
}

// EpochField impl common.Document
func (eg *ExecGas) EpochField() *string {
	return &execGasEpochField
}

// ResetPolicy impls common.Document
func (eg *ExecGas) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(execGasEpochField, lower, upper), true
}
