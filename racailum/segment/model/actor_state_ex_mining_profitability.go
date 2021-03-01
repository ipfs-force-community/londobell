package model

import (
	"github.com/dtynn/londobell/common"
	"github.com/filecoin-project/go-state-types/abi"
)

var (
	_ common.IndexedDocument = (*MiningProfitability)(nil)

	miningProfitabilityColName    = getColName(MiningProfitability{})
	miningProfitabilityEpochField = extractEpochFieldName(MiningProfitability{})
)

// MiningProfitabilityDetail contains pledge & projection for 32GiB QA power
type MiningProfitabilityDetail struct {
	ExpectedDayReward         abi.TokenAmount
	InitialPledge             abi.TokenAmount
	InitialConsensusPledge    abi.TokenAmount
	InitialStoragePledge      abi.TokenAmount
	ProjectionOfInitialPledge abi.TokenAmount
	ProjectionOfFaultFee      abi.TokenAmount
}

// MiningProfitability shows profitability for mining issues
type MiningProfitability struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MiningProfitabilityDetail
}

// CollectionName impl common.Document
func (m *MiningProfitability) CollectionName() string {
	return miningProfitabilityColName
}

// EpochField impl common.Document
func (m *MiningProfitability) EpochField() *string {
	return &miningProfitabilityEpochField
}

// ResetPolicy impl common.Document
func (m *MiningProfitability) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(miningProfitabilityEpochField, lower, upper), true
}
