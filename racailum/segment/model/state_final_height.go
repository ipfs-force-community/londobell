package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*StateFinalHeight)(nil)

	stateFinalHeightColName    = getColName(StateFinalHeight{})
	stateFinalHeightEpochField = extractEpochFieldName(StateFinalHeight{})
)

type StateFinalHeight struct {
	Epoch abi.ChainEpoch `bson:"_id"`
	Cids  []cid.Cid
}

func (f *StateFinalHeight) Indexes() [][]string {
	return [][]string{
		{"Cids"}, //
	}
}

func NewStateFinalHeight(ts *common.LinkedTipSet) (*StateFinalHeight, error) {
	return &StateFinalHeight{
		Epoch: ts.Height(),
		Cids:  ts.Key().Cids(),
	}, nil
}

// CollectionName impl common.Document
func (f *StateFinalHeight) CollectionName() string {
	return stateFinalHeightColName
}

// EpochField impl common.Document
func (f *StateFinalHeight) EpochField() *string {
	return &stateFinalHeightEpochField
}

// ResetPolicy impl common.Document
func (f *StateFinalHeight) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(finalHeightEpochField, lower, upper), true
}
