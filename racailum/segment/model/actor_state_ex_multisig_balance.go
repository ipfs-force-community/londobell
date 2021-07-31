package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/dtynn/londobell/common"
)

var (
	_ common.IndexedDocument = (*MultisigBalance)(nil)

	multisigBalanceColName    = getColName(MultisigBalance{})
	multisigBalanceEpochField = extractEpochFieldName(MultisigBalance{})
)

// MultisigBalanceDetail is the detail format of the MultisigBalance
type MultisigBalanceDetail struct {
	Locked       abi.TokenAmount
	Vested       abi.TokenAmount
	VestInFuture []abi.TokenAmount
}

// MultisigBalance shows the balance of a specific multisig actor
type MultisigBalance struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MultisigBalanceDetail
}

// CollectionName impl common.Document
func (c *MultisigBalance) CollectionName() string {
	return multisigBalanceColName
}

// EpochField impl common.Document
func (c *MultisigBalance) EpochField() *string {
	return &multisigBalanceEpochField
}

// ResetPolicy impl common.Document
func (c *MultisigBalance) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(multisigBalanceEpochField, lower, upper), true
}
