package model

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mir"
)

var (
	_ common.IndexedDocument = (*BlockHeader)(nil)

	blockHeaderColName    = getColName(BlockHeader{})
	blockHeaderEpochField = extractEpochFieldName(BlockHeader{})
)

// BlockHeader contains required fields from *types.BlockHeader, most of which are unique per block/miner
type BlockHeader struct {
	ID            cid.Cid `mir:"-" bson:"_id" `
	Miner         address.Address
	Epoch         abi.ChainEpoch `mir:"Height"`
	Messages      cid.Cid
	ElectionProof *types.ElectionProof
	Ticket        *types.Ticket
	MessageCount  int
}

// NewBlockHeader takes required fields from a raw *types.BlockHeader
func NewBlockHeader(minerID address.Address, raw *types.BlockHeader) (*BlockHeader, error) {
	bh := &BlockHeader{
		ID: raw.Cid(),
	}

	if err := mir.Mirror(bh, raw); err != nil {
		return nil, fmt.Errorf("mirroring BlockHeader: %w", err)
	}

	bh.Miner = minerID
	return bh, nil
}

// Indexes impl common.Indexed
func (bh *BlockHeader) Indexes() [][]string {
	return [][]string{
		[]string{blockHeaderEpochField, "Miner"},
	}
}

// CollectionName impl common.Document
func (bh *BlockHeader) CollectionName() string {
	return blockHeaderColName
}

// EpochField impl common.Document
func (bh *BlockHeader) EpochField() *string {
	return &blockHeaderEpochField
}

// ResetPolicy impl common.Document
func (bh *BlockHeader) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(blockHeaderEpochField, lower, upper), true
}

func (bh *BlockHeader) IsMutable() bool {
	return false
}
