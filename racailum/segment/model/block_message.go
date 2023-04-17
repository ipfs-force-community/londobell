package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                      common.IndexedDocument = (*BlockMessage)(nil)
	blockMessageEpochField                        = extractEpochFieldName(BlockMessage{})
	blockMessageColName                           = getColName(BlockMessage{})
)

// BlockMessage records map of block cid to message cids
type BlockMessage struct {
	Cid      cid.Cid        `bson:"_id" `
	Epoch    abi.ChainEpoch `mir:"-"`
	Messages []cid.Cid
}

func NewBlockMessage(bcid cid.Cid, epoch abi.ChainEpoch, mcids []cid.Cid) (*BlockMessage, error) {
	return &BlockMessage{
		Cid:      bcid,
		Epoch:    epoch,
		Messages: mcids,
	}, nil
}

// Indexes impl common.Indexed
func (bm *BlockMessage) Indexes() [][]string {
	return [][]string{
		[]string{blockMessageEpochField, "_id"},
		[]string{blockMessageEpochField, "Messages"},
	}
}

// CollectionName impl common.Document
func (bm *BlockMessage) CollectionName() string {
	return blockMessageColName
}

// EpochField impl common.Document
func (bm *BlockMessage) EpochField() *string {
	return &blockMessageEpochField
}

// ResetPolicy impl common.Document
func (bm *BlockMessage) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(blockMessageEpochField, lower, upper), true
}
