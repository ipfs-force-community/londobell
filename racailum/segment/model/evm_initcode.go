package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*EvmInitCode)(nil)

	evmInitCodeColName    = getColName(EvmInitCode{})
	evmInitCodeEpochField = extractEpochFieldName(EvmInitCode{})
)

// NewExecTrace converts raw exec trace struct to ExecTrace*
func NewEvmInitCode(
	actorID address.Address,
	initCode string,
	epoch abi.ChainEpoch,
) *EvmInitCode {
	return &EvmInitCode{
		ActorID:  actorID,
		InitCode: initCode,
		Epoch:    epoch,
	}
}

// ExecTrace is the schema of *api.ExecutionTrace
type EvmInitCode struct {
	ActorID  address.Address `mir:"-" bson:"_id"`
	InitCode string
	Epoch    abi.ChainEpoch
}

// EvmByteCode impl common.Indexed
func (ei *EvmInitCode) Indexes() [][]string {
	return [][]string{
		[]string{evmInitCodeEpochField},
	}
}

// CollectionName impls common.Document
func (ei *EvmInitCode) CollectionName() string {
	return evmInitCodeColName
}

// EpochField impl common.Document
func (ei *EvmInitCode) EpochField() *string {
	return &evmInitCodeEpochField
}

// ResetPolicy impls common.Document
func (ei *EvmInitCode) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(evmInitCodeEpochField, lower, upper), true
}

func (ei *EvmInitCode) IsMutable() bool {
	return false
}
