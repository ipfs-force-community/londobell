package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                      common.IndexedDocument = (*MessageBlock)(nil)
	messageBlockEpochField                        = extractEpochFieldName(MessageBlock{})
	messageBlockColName                           = getColName(MessageBlock{})
)

// MessageBlock records map of message cid to block cid
type MessageBlock struct {
	Cid    cid.Cid        `bson:"_id" `
	Epoch  abi.ChainEpoch `mir:"-"`
	Blocks []cid.Cid
}

func NewMessageBlock(mcid cid.Cid, epoch abi.ChainEpoch, bcids []cid.Cid) (*MessageBlock, error) {
	return &MessageBlock{
		Cid:    mcid,
		Epoch:  epoch,
		Blocks: bcids,
	}, nil
}

// Indexes impl common.Indexed
func (bh *MessageBlock) Indexes() [][]string {
	return [][]string{
		[]string{messageBlockEpochField, "Blocks"},
	}
}

// CollectionName impl common.Document
func (bh *MessageBlock) CollectionName() string {
	return messageBlockColName
}

// EpochField impl common.Document
func (bh *MessageBlock) EpochField() *string {
	return &messageBlockEpochField
}

// ResetPolicy impl common.Document
func (bh *MessageBlock) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(messageBlockEpochField, lower, upper), true
}
