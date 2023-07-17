package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.Document = (*FilSupply)(nil)

	filSupplyColName    = getColName(FilSupply{})
	filSupplyEpochField = extractEpochFieldName(FilSupply{})
)

// FilSupply shows the summary of the fil circulating supply
type FilSupply struct {
	Epoch abi.ChainEpoch `bson:"_id"`
	api.CirculatingSupply
}

// CollectionName impl common.Document
func (f *FilSupply) CollectionName() string {
	return filSupplyColName
}

// EpochField impl common.Document
func (f *FilSupply) EpochField() *string {
	return &filSupplyEpochField
}

// ResetPolicy impl common.Document
func (f *FilSupply) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(filSupplyEpochField, lower, upper), true
}

func (f *FilSupply) IsMutable() bool {
	return false
}
