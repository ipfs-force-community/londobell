package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*AllocatedSectors)(nil)

	allocatedSectorsColName    = getColName(AllocatedSectors{})
	allocatedSectorsEpochField = extractEpochFieldName(AllocatedSectors{})
)

// AllocatedSectorsDetail is a type alias for BitfieldDetail
type AllocatedSectorsDetail = BitfieldDetail

// AllocatedSectors is the rle runs generated from the AllocatedSectors field in *miner.State
type AllocatedSectors struct {
	ActorStateExBasic `bson:",inline"`
	Detail            AllocatedSectorsDetail
}

// CollectionName impl common.Document
func (c *AllocatedSectors) CollectionName() string {
	return allocatedSectorsColName
}

// EpochField impl common.Document
func (c *AllocatedSectors) EpochField() *string {
	return &allocatedSectorsEpochField
}

// ResetPolicy impl common.Document
func (c *AllocatedSectors) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(allocatedSectorsEpochField, lower, upper), true
}
