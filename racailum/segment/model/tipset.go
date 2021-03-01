package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/dtynn/londobell/common"
)

var _ common.Document = (*TipSet)(nil)

var (
	tipsetColName    = getColName(TipSet{})
	tipsetEpochField = extractEpochFieldName(TipSet{})
)

// NewTipSet converts a *common.LinkedTipSet to *TipSet
func NewTipSet(ts *common.LinkedTipSet) (*TipSet, error) {
	minChildBlk := ts.Child.MinTicketBlock()
	return &TipSet{
		Epoch:        ts.Height(),
		Cids:         ts.Key().Cids(),
		MinTimestamp: ts.MinTimestamp(),
		ChildEpoch:   ts.Child.Height(),
		State:        ts.State(),
		Receipts:     minChildBlk.ParentMessageReceipts,
		Weight:       ts.Child.ParentWeight(),
		BaseFee:      minChildBlk.ParentBaseFee,
	}, nil
}

// TipSet contains the basic info about a tipset on the heaviest chain
type TipSet struct {
	Epoch        abi.ChainEpoch `bson:"_id"`
	Cids         []cid.Cid
	MinTimestamp uint64
	ChildEpoch   abi.ChainEpoch
	State        cid.Cid
	Receipts     cid.Cid
	Weight       types.BigInt
	BaseFee      abi.TokenAmount
}

// CollectionName impl common.Document
func (t *TipSet) CollectionName() string {
	return "Tipset"
}

// EpochField impl common.Document
func (t *TipSet) EpochField() *string {
	return &tipsetEpochField
}

// ResetPolicy impl common.Document
func (t *TipSet) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(tipsetEpochField, lower, upper), true
}
