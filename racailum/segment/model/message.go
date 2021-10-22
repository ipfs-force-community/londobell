package model

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mgoutil/mcodec"
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
	Cid            cid.Cid `bson:"_id"`
	*types.Message `bson:",inline"`
	Detail         MessageDetail
}

// Indexes impl common.Indexed
func (m *Message) Indexes() [][]string {
	return [][]string{
		[]string{"From", "Nonce"},
		[]string{"To", "Method"},
		[]string{"Detail.Method", "Detail.Actor"},
		[]string{"Detail.PackedHeight"},
		[]string{"Detail.PackedHeight", "Detail.Method"},
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

// NewMessage converts from *types.Message to *Message with required infomations
func NewMessage(mcid cid.Cid, raw *types.Message, act, meth string, params cbor.Er, epoch abi.ChainEpoch) (*Message, error) {
	msg := &Message{
		Cid:     mcid,
		Message: raw,
	}

	msg.Detail.Actor = act
	msg.Detail.Method = meth

	if params != nil && len(raw.Params) > 0 {
		err := params.UnmarshalCBOR(bytes.NewReader(raw.Params))
		if err != nil {
			return nil, fmt.Errorf("unmarshal cbor for message params: %w", err)
		}

		msg.Detail.Params = params
	}

	msg.Detail.PackedHeight = epoch

	return msg, nil
}
