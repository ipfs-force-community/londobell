package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*VerifiedRegistry)(nil)

	verifiedRegistryColName    = getColName(VerifiedRegistry{})
	verifiedRegistryEpochField = extractEpochFieldName(VerifiedRegistry{})
)

// VerifiedRegistryDetail contains type & cap info
type VerifiedRegistryDetail struct {
	Type string
	Cap  abi.StoragePower
}

// VerifiedRegistry is the details of data capacity of notaries or clients
type VerifiedRegistry struct {
	ActorStateExBasic `bson:",inline"`
	Detail            VerifiedRegistryDetail
}

// CollectionName impl common.Document
func (c *VerifiedRegistry) CollectionName() string {
	return verifiedRegistryColName
}

// EpochField impl common.Document
func (c *VerifiedRegistry) EpochField() *string {
	return &verifiedRegistryEpochField
}

// ResetPolicy impl common.Document
func (c *VerifiedRegistry) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(verifiedRegistryEpochField, lower, upper), true
}
