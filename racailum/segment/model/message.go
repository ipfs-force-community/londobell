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
	Cid            cid.Cid `bson:"_id"`
	*types.Message `bson:",inline"`
	Detail         MessageDetail
	SignedCid      cid.Cid `bson:"SignedCid,omitempty"`
}

// Indexes impl common.Indexed
func (m *Message) Indexes() [][]string {
	return [][]string{
		[]string{"SignedCid"},
		[]string{"Detail.Method", "Detail.PackedHeight", "Detail.Params.Deals.Proposal.Provider"},
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
func NewMessage(mcid, signedCid cid.Cid, raw *types.Message, act, meth string, params cbor.Er, epoch abi.ChainEpoch) (*Message, error) {
	msg := &Message{
		Cid:       mcid,
		Message:   raw,
		SignedCid: signedCid,
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
