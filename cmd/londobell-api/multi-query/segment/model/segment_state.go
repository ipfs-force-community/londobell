package model

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var _ common.IndexedDocument = (*SegmentState)(nil)

type SegmentState struct {
	ID         string `bson:"_id"`
	Dsn        string
	StartEpoch abi.ChainEpoch // included
	EndEpoch   abi.ChainEpoch // not included
	Count      int64

	ActorID    string
	MethodName string

	AllMethodNames []string
}

func NewSegmentState(dsn string, startEpoch, endEpoch abi.ChainEpoch, count int64) *SegmentState {
	return &SegmentState{
		ID:         fmt.Sprintf("%v-%v", dsn, startEpoch),
		Dsn:        dsn,
		StartEpoch: startEpoch,
		EndEpoch:   endEpoch,
		Count:      count,
	}
}

// Indexes impl common.Indexed
func (s *SegmentState) Indexes() [][]string {
	return [][]string{
		{"StartEpoch"},
	}
}

// CollectionName impl common.Document
func (s *SegmentState) CollectionName() string {
	return "SegmentState"
}

// EpochField impl common.Document
func (s *SegmentState) EpochField() *string {
	return nil
}

// ResetPolicy impl common.Document
func (s *SegmentState) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return nil, false
}

func (s *SegmentState) IsMutable() bool {
	return false
}
