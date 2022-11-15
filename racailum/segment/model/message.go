package model

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

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

	if len(raw.Params) > 0 {
		if meth == "InvokeContract" && strings.Contains(act, "evm") {
			// parse contract method
			hexParams, err := hexEncodeParams(raw.Params)
			if err != nil {
				return nil, fmt.Errorf("hex encode params failed: %w", err)
			}

			// the first 4 bytes is method ID
			if len(hexParams) < 8 {
				return nil, fmt.Errorf("invalid length of params %v for InvokeContract", len(hexParams))
			}
			if (len(hexParams)-8)%64 != 0 {
				return nil, fmt.Errorf("invalid length of params %v for InvokeContract", len(hexParams))
			}

			//methodID := hexParams[:8]
			//params := hexParams[8:]
			// methodID has been recorded
			msg.Detail.Params = hexString(hexParams)
		}

	}

	msg.Detail.PackedHeight = epoch

	return msg, nil
}

type hexString string

func (h hexString) MarshalCBOR(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func (h hexString) UnmarshalCBOR(r io.Reader) error {
	//TODO implement me
	panic("implement me")
}

func hexEncodeParams(params []byte) (string, error) {
	buffer := bytes.NewBuffer(params)
	hexParams, err := cbg.ReadByteArray(buffer, 1024)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hexParams), nil
}
