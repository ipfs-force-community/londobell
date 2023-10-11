package model

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

var (
	_                      common.IndexedDocument = (*ActorMessage)(nil)
	actorMessageEpochField                        = extractEpochFieldName(ActorMessage{})
	actorMessageColName                           = getColName(ActorMessage{})
)

const (
	Null        = ""
	Blockreward = "Blockreward"
	Burn        = "Burn"
	Send        = "Send"
	Receive     = "Receive"
)

// ActorMessage records messages for actor
type ActorMessage struct {
	ID            string `mir:"-" bson:"_id"`
	ActorID       address.Address
	Epoch         abi.ChainEpoch `mir:"-"`
	Cid           cid.Cid
	SignedCid     cid.Cid
	Value         abi.TokenAmount // int64
	MethodName    string
	ExitCode      exitcode.ExitCode
	Type          string // "from" or "to"
	From          address.Address
	To            address.Address
	IsBlock       bool    // 是否是块消息
	TransferType  string  // "Blockreward", "Burn", "Send", "Receive"
	RootCid       cid.Cid `mir:"-"`
	RootSignedCid cid.Cid `mir:"-"`
}

func NewActorMessage(ctx *extract.Ctx, actorID address.Address, epoch abi.ChainEpoch, cid, signedCid cid.Cid, value abi.TokenAmount, methodName string, exitcode exitcode.ExitCode, mtype string, from, to address.Address, isBlock bool, seq []int, transferType string, IDCidMap map[string][2]cid.Cid) (*ActorMessage, error) {
	elog := ctx.L.With("NewActorMessage", cid)
	am := &ActorMessage{
		ActorID:      actorID,
		Epoch:        epoch,
		Cid:          cid,
		SignedCid:    signedCid,
		Value:        value,
		MethodName:   methodName,
		ExitCode:     exitcode,
		Type:         mtype,
		From:         from,
		To:           to,
		IsBlock:      isBlock,
		TransferType: transferType,
	}

	am.genID(epoch, mtype, seq)
	err := am.genRootids(IDCidMap)
	if err != nil {
		elog.Warn(err)
	}
	return am, nil
}

// Indexes impl common.Indexed
func (am *ActorMessage) Indexes() [][]string {
	return [][]string{
		//[]string{"ActorID", "IsBlock", actorMessageEpochField}, // todo: 用ActorID_1_IsBlock_1_MethodName_1_Epoch_1？
		[]string{"ActorID", "IsBlock", "MethodName", actorMessageEpochField},
		[]string{"ActorID", "ExitCode", "TransferType", actorMessageEpochField},
		[]string{"ActorID", "ExitCode", "Value", actorMessageEpochField},
		[]string{"IsBlock", "Type", actorMessageEpochField},
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

func (am *ActorMessage) genID(epoch abi.ChainEpoch, mtype string, seq []int) {
	seqStrs := make([]string, 0, len(seq))
	for i := range seq {
		seqStrs = append(seqStrs, fmt.Sprintf("%05d", seq[i]))
	}

	am.ID = fmt.Sprintf("%d-%s-%s", epoch, strings.Join(seqStrs, "-"), mtype)
}

func (am *ActorMessage) IsMutable() bool {
	return false
}

// get root Cid SignedCid
func (am *ActorMessage) genRootids(m map[string][2]cid.Cid) error {
	if am.IsBlock {
		return nil
	}
	subs := strings.Split(am.ID, "-")
	if len(subs) < 2 {
		return fmt.Errorf("getRootids Split length err: %s", am.ID)
	}
	rootID := subs[0] + "-" + subs[1]
	am.RootCid = m[rootID][0]
	am.RootSignedCid = m[rootID][1]
	return nil
}
