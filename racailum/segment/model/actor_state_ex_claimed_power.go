package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mgoutil/mcodec"
)

func init() {
	mcodec.RegisterSchemaType(new(ClaimedPowerDetail))
}

var (
	_ common.IndexedDocument = (*ClaimedPower)(nil)

	claimedPowerColName    = getColName(ClaimedPower{})
	claimedPowerEpochField = extractEpochFieldName(ClaimedPower{})
)

// ClaimedPowerDetail is a type alias for cbor.Er
type ClaimedPowerDetail cbor.Er

// ClaimedPower is the power info extracted from field `Claims`
type ClaimedPower struct {
	ActorStateExBasic `bson:",inline"`
	Detail            ClaimedPowerDetail
}

// CollectionName impl common.Document
func (c *ClaimedPower) CollectionName() string {
	return claimedPowerColName
}

// EpochField impl common.Document
func (c *ClaimedPower) EpochField() *string {
	return &claimedPowerEpochField
}

// ResetPolicy impl common.Document
func (c *ClaimedPower) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(claimedPowerEpochField, lower, upper), true
}
