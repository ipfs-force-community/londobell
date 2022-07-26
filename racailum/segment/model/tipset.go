package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

var _ common.IndexedDocument = (*TipSet)(nil)

var (
	// tipsetColName    = getColName(TipSet{})
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

func NewTipSetWithoutChild(ts *common.LinkedTipSet, weight types.BigInt, baseFee abi.TokenAmount, st cid.Cid) (*TipSet, error) {
	return &TipSet{
		Epoch:        ts.Height(),
		Cids:         ts.Key().Cids(),
		MinTimestamp: ts.MinTimestamp(),
		//ChildEpoch:   0,
		State:    st,
		Receipts: cid.Undef, //
		Weight:   weight,
		BaseFee:  baseFee,
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

// Indexes impl common.Indexed
func (t *TipSet) Indexes() [][]string {
	return [][]string{
		{"ChildEpoch"},
	}
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
