package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                         common.IndexedDocument = (*DatacapBalances)(nil)
	datacapBalancesColName                           = getColName(DatacapBalances{})
	datacapBalancesEpochField                        = extractEpochFieldName(DatacapBalances{})
)

type DatacapBalancesDetail struct {
	Owner  abi.ActorID
	Amount abi.TokenAmount
}

type DatacapBalances struct {
	ID     cid.Cid `bson:"_id"`
	Epoch  abi.ChainEpoch
	Owner  abi.ActorID
	Amount abi.TokenAmount
}

func (a *DatacapBalances) CollectionName() string {
	return datacapBalancesColName
}

func (a *DatacapBalances) EpochField() *string {
	return &datacapBalancesEpochField
}

func (a *DatacapBalances) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(datacapBalancesEpochField, lower, upper), true
}

func (a *DatacapBalances) Indexes() [][]string {
	return [][]string{
		[]string{"Owner"},
		[]string{datacapBalancesEpochField, "Owner"},
	}
}

func (a *DatacapBalances) IsMutable() bool {
	return false
}
