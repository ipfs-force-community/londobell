package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/dtynn/londobell/common"
)

var (
	_ common.IndexedDocument = (*MinerFunds)(nil)

	marketFundsDetailColName    = getColName(MarketFunds{})
	marketFundsDetailEpochField = extractEpochFieldName(MarketFunds{})
)

// MarketFundsDetail contains several token amounts
type MarketFundsDetail struct {
	TotalLocked abi.TokenAmount `mir:"-"`

	TotalClientLockedCollateral   abi.TokenAmount
	TotalProviderLockedCollateral abi.TokenAmount
	TotalClientStorageFee         abi.TokenAmount

	ClientUnLockCollateralInFuture   []abi.TokenAmount `mir:"-"`
	ProviderUnLockCollateralInFuture []abi.TokenAmount `mir:"-"`
	ClientUnlockStorageFeeInFuture   []abi.TokenAmount `mir:"-"`
}

// MarketFunds shows funding details for miner
type MarketFunds struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MarketFundsDetail
}

// CollectionName impl common.Document
func (m *MarketFunds) CollectionName() string {
	return marketFundsDetailColName
}

// EpochField impl common.Document
func (m *MarketFunds) EpochField() *string {
	return &marketFundsDetailEpochField
}

// ResetPolicy impl common.Document
func (m *MarketFunds) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(marketFundsDetailEpochField, lower, upper), true
}
