package model

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var _ common.IndexedDocument = (*SegmentDealState)(nil)

type SegmentDealState struct {
	ID          string `bson:"_id"`
	Dsn         string
	StartDealID uint64 // included
	EndDealID   uint64 // not included
	Count       int64

	ActorID string
}

func NewSegmentDealState(dsn string, startDealID, endDealID uint64, count int64) *SegmentDealState {
	return &SegmentDealState{
		ID:          fmt.Sprintf("%v-%v", dsn, startDealID),
		Dsn:         dsn,
		StartDealID: startDealID,
		EndDealID:   endDealID,
		Count:       count,
	}
}

// Indexes impl common.Indexed
func (s *SegmentDealState) Indexes() [][]string {
	return [][]string{
		{"StartDealID"},
	}
}

// CollectionName impl common.Document
func (s *SegmentDealState) CollectionName() string {
	return "SegmentDealState"
}

// EpochField impl common.Document
func (s *SegmentDealState) EpochField() *string {
	return nil
}

// ResetPolicy impl common.Document
func (s *SegmentDealState) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return nil, false
}
