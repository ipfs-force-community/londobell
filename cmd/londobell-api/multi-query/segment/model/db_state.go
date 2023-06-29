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

const (
	Tmp DType = iota
	Formal
	Cold
)

type DBState struct {
	ID    string `bson:"_id"`
	Dsn   string
	DType DType // formal or tmp or cold

	Interval abi.ChainEpoch // 间隔

	StartEpoch abi.ChainEpoch // 整个库查询的开始
	EndEpoch   abi.ChainEpoch // finalHeight+1, 右开
}

func NewDBState(dsn string, dtype DType, interval, startEpoch, endEpoch abi.ChainEpoch) *DBState {
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

func DefaultDBState(dsn string, dtype DType, start, end, interval abi.ChainEpoch) *DBState {
	return &DBState{
		Dsn:        dsn,
		DType:      dtype,
		Interval:   interval,
		StartEpoch: start,
		EndEpoch:   end,
	}
}

func (d *DBState) GetDType() DType {
	return d.DType
}
