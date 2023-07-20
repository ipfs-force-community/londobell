package model

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var _ common.IndexedDocument = (*DealState)(nil)

const NoneInterval = -1

type DealState struct {
	ID    string `bson:"_id"`
	Dsn   string
	DType DType // formal or tmp or cold

	Interval int64 // 间隔 -1表示不分组

	StartDealID uint64 // included
	EndDealID   uint64 // not included

	Count int64
}

func NewDealState(dsn string, dtype DType, interval int64, startDealID, endDealID uint64, count int64) *DealState {
	return &DealState{
		ID:          fmt.Sprintf("%v-%v", dsn, startDealID),
		Dsn:         dsn,
		DType:       dtype,
		StartDealID: startDealID,
		EndDealID:   endDealID,
		Count:       count,
		Interval:    interval,
	}
}

// Indexes impl common.Indexed
func (s *DealState) Indexes() [][]string {
	return [][]string{
		{"StartDealID"},
	}
}

// CollectionName impl common.Document
func (s *DealState) CollectionName() string {
	return "DealState"
}

// EpochField impl common.Document
func (s *DealState) EpochField() *string {
	return nil
}

// ResetPolicy impl common.Document
func (s *DealState) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return nil, false
}

func (s *DealState) IsMutable() bool {
	return false
}
