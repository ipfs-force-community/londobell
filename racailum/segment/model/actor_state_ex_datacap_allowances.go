package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs/go-cid"
)

var (
	_                           common.IndexedDocument = (*DatacapAllowances)(nil)
	datacapAllowancesColName                           = getColName(DatacapAllowances{})
	datacapAllowancesEpochField                        = extractEpochFieldName(DatacapAllowances{})
)

type DatacapAllowances struct {
	ID       cid.Cid `bson:"_id"`
	Epoch    abi.ChainEpoch
	Owner    abi.ActorID
	Operator abi.ActorID
	Amount   abi.TokenAmount
}

func (a *DatacapAllowances) CollectionName() string {
	return datacapAllowancesColName
}

func (a *DatacapAllowances) EpochField() *string {
	return &datacapAllowancesEpochField
}

func (a *DatacapAllowances) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(datacapAllowancesEpochField, lower, upper), true
}

func (a *DatacapAllowances) Indexes() [][]string {
	return [][]string{
		[]string{"Owner"},
		[]string{datacapAllowancesEpochField, "Owner"},
		[]string{datacapAllowancesEpochField, "Owner", "Operator"},
	}
}

func (a *DatacapAllowances) IsMutable() bool {
	return false
}
