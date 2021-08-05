package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/dtynn/londobell/common"
)

var (
	_ common.IndexedDocument = (*MinerFunds)(nil)

	minerFundsColName    = getColName(MinerFunds{})
	minerFundsEpochField = extractEpochFieldName(MinerFunds{})
)

// MinerFundsDetail contains several token amounts
type MinerFundsDetail struct {
	PreCommitDeposits abi.TokenAmount

	VestingTotal  abi.TokenAmount `mir:"-"`
	LockedFunds   abi.TokenAmount
	FeeDebt       abi.TokenAmount
	InitialPledge abi.TokenAmount

	VestInFuture  []abi.TokenAmount
	PledgeRelease []abi.TokenAmount
}

// MinerFunds shows funding details for miner
type MinerFunds struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MinerFundsDetail
}

// CollectionName impl common.Document
func (m *MinerFunds) CollectionName() string {
	return minerFundsColName
}

// EpochField impl common.Document
func (m *MinerFunds) EpochField() *string {
	return &minerFundsEpochField
}

// ResetPolicy impl common.Document
func (m *MinerFunds) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerFundsEpochField, lower, upper), true
}
