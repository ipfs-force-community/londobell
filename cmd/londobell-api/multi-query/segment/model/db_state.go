package model

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin"

	"github.com/ipfs-force-community/londobell/common"
)

var _ common.IndexedDocument = (*DBState)(nil)

const DefaultInterval = 3 * builtin.EpochsInHour

type DType int

func (dt DType) ToString() string {
	switch dt {
	case Tmp:
		return "tmp"
	case Formal:
		return "formal"
	default:
		return "cold"
	}
}

const (
	Tmp DType = iota
	Formal
	Cold
)

type DBState struct {
	ID    string `bson:"_id"`
	Dsn   string
	DType DType // formal or tmp or cold

	Interval int64 // 高度间隔

	StartEpoch abi.ChainEpoch // 整个库查询的开始
	EndEpoch   abi.ChainEpoch // finalHeight+1, 右开
}

func NewDBState(dsn string, dtype DType, interval int64, startEpoch, endEpoch abi.ChainEpoch) *DBState {
	return &DBState{
		ID:         fmt.Sprintf("%v-%v", dsn, startEpoch),
		Dsn:        dsn,
		DType:      dtype,
		Interval:   interval,
		StartEpoch: startEpoch,
		EndEpoch:   endEpoch,
	}
}

// Indexes impl common.Indexed
func (d *DBState) Indexes() [][]string {
	return [][]string{
		{"DType", "StartEpoch"},
	}
}

// CollectionName impl common.Document
func (d *DBState) CollectionName() string {
	return "DBState"
}

// EpochField impl common.Document
func (d *DBState) EpochField() *string {
	return nil
}

// ResetPolicy impl common.Document
func (d *DBState) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return nil, false
}

func (d *DBState) GetDType() DType {
	return d.DType
}

func (d *DBState) IsMutable() bool {
	return false
}
