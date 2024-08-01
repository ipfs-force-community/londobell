package model

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"
)

func init() {
	mcodec.RegisterSchemaType(new(MessageParams))
}

var _ common.IndexedDocument = (*Message)(nil)

var (
	messageColName = getColName(Message{})
)

// MessageParams is a type alias
type MessageParams cbor.Er

// MessageDetail is the detail of message
type MessageDetail struct {
	Actor        string
	Method       string
	Params       MessageParams
	PackedHeight abi.ChainEpoch
}

// Message is the schema of *types.Message
type Message struct {
	Cid        cid.Cid             `bson:"_id"`
	Message    *types.MessageTrace `bson:",inline"`
	Nonce      uint64
	Detail     MessageDetail
	SignedCid  cid.Cid `bson:"SignedCid,omitempty"`
	GasFeeCap  abi.TokenAmount
	GasPremium abi.TokenAmount
}

// Indexes impl common.Indexed
func (m *Message) Indexes() [][]string {
	return [][]string{
		[]string{"From", "Nonce"},
		[]string{"To", "Method"},
		[]string{"Detail.Method", "Detail.Actor"},
		[]string{"Detail.PackedHeight"},
		[]string{"Detail.PackedHeight", "Detail.Method"},
		[]string{"SignedCid"},
	}
}

// CollectionName impl common.Document
func (m *Message) CollectionName() string {
	return messageColName
}

// EpochField impl common.Document
func (m *Message) EpochField() *string {
	return nil
}

// ResetPolicy impl common.Document
func (m *Message) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return nil, false
}

func (m *Message) IsMutable() bool {
	return false
}

// NewMessage converts from *types.Message to *Message with required infomations
func NewMessage(mcid, signedCid cid.Cid, raw *types.MessageTrace, act, meth string, params cbor.Er,
	epoch abi.ChainEpoch, rootMsg *types.Message, rootCid cid.Cid, nonce uint64, isBlock bool) (*Message, error) {
	var msg *Message
	msgTrace := raw

	// If MessageTrace is root message, then use Message.GasLimit value
	if mcid.Equals(rootCid) {
		rawClone := *raw
		rawClone.GasLimit = uint64(rootMsg.GasLimit)
		msgTrace = &rawClone
	}
	if isBlock {
		msg = &Message{
			Cid:        mcid,
			Message:    msgTrace,
			SignedCid:  signedCid,
			GasPremium: rootMsg.GasPremium,
			GasFeeCap:  rootMsg.GasFeeCap,
			Nonce:      nonce,
		}
	} else {
		msg = &Message{
			Cid:        mcid,
			Message:    msgTrace,
			SignedCid:  signedCid,
			GasPremium: rootMsg.GasPremium,
			GasFeeCap:  rootMsg.GasFeeCap,
		}
	}

	msg.Detail.Actor = act
	msg.Detail.Method = meth

	if params != nil && len(raw.Params) > 0 {
		err := params.UnmarshalCBOR(bytes.NewReader(raw.Params))
		if err != nil {
			return nil, fmt.Errorf("unmarshal cbor for message params, codec %v,err: %w", raw.ParamsCodec, err)
		}

		msg.Detail.Params = params
	}

	msg.Detail.PackedHeight = epoch

	return msg, nil
}
