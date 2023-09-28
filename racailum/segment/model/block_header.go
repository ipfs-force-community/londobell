package model

import (
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/proof"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mir"
)

var (
	_ common.IndexedDocument = (*BlockHeader)(nil)

	blockHeaderColName    = getColName(BlockHeader{})
	blockHeaderEpochField = extractEpochFieldName(BlockHeader{})
	// block 1st seen cache : make cache with 1hour TTL and 10000 max keys
	b1sCache = expirable.NewLRU[cid.Cid, int64](10000, nil, time.Hour*1)
)

// BlockHeader contains required fields from *types.BlockHeader, most of which are unique per block/miner
type BlockHeader struct {
	ID                    cid.Cid `mir:"-" bson:"_id" `
	Miner                 address.Address
	Epoch                 abi.ChainEpoch `mir:"Height"`
	Messages              cid.Cid
	ElectionProof         *types.ElectionProof
	Ticket                *types.Ticket
	BeaconEntries         []types.BeaconEntry
	WinPoStProof          []proof.PoStProof
	Parents               []cid.Cid
	ParentWeight          types.BigInt
	ParentStateRoot       cid.Cid
	ParentMessageReceipts cid.Cid
	BLSAggregate          *crypto.Signature
	Timestamp             uint64
	BlockSig              *crypto.Signature
	ForkSignaling         uint64
	ParentBaseFee         abi.TokenAmount
	MessageCount          int
	// fist seen timestamp
	FirstSeen int64
}

// NewBlockHeader takes required fields from a raw *types.BlockHeader
func NewBlockHeader(minerID address.Address, raw *types.BlockHeader) (*BlockHeader, error) {
	bh := &BlockHeader{
		ID: raw.Cid(),
	}

	if err := mir.Mirror(bh, raw); err != nil {
		return nil, fmt.Errorf("mirroring BlockHeader: %w", err)
	}
	if bst, ok := b1sCache.Get(bh.ID); ok {
		bh.FirstSeen = bst
	} else {
		now := time.Now().Unix()
		b1sCache.Add(bh.ID, now)
		bh.FirstSeen = now
	}
	bh.Miner = minerID
	return bh, nil
}

// Indexes impl common.Indexed
func (bh *BlockHeader) Indexes() [][]string {
	return [][]string{
		[]string{blockHeaderEpochField, "Miner"},
		[]string{"Miner", blockHeaderEpochField},
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
