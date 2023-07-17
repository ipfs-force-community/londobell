package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*FinalHeight)(nil)

	finalHeightColName    = getColName(FinalHeight{})
	finalHeightEpochField = extractEpochFieldName(FinalHeight{})
)

type FinalHeight struct {
	Epoch abi.ChainEpoch `bson:"_id"`
	Cids  []cid.Cid
}

func (f *FinalHeight) Indexes() [][]string {
	return [][]string{
		{"Cids"}, //
	}
}

func NewFinalHeight(ts *common.LinkedTipSet) (*FinalHeight, error) {
	return &FinalHeight{
		Epoch: ts.Height(),
		Cids:  ts.Key().Cids(),
	}, nil
}

// CollectionName impl common.Document
func (f *FinalHeight) CollectionName() string {
	return finalHeightColName
}

// EpochField impl common.Document
func (f *FinalHeight) EpochField() *string {
	return &finalHeightEpochField
}

// ResetPolicy impl common.Document
func (f *FinalHeight) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(finalHeightEpochField, lower, upper), true
}

func (f *FinalHeight) IsMutable() bool {
	return false
}
