package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                     common.IndexedDocument = (*PendingTxns)(nil)
	pendingTxnsColName                           = getColName(PendingTxns{})
	pendingTxnsEpochField                        = extractEpochFieldName(PendingTxns{})
)

type PendingTxnsDetail struct {
	TxnID    int64
	To       address.Address
	Value    abi.TokenAmount
	Params   []byte
	Approved []address.Address
}

type PendingTxns struct {
	ActorStateExBasic `bson:",inline"`
	Detail            PendingTxnsDetail
}

func (p *PendingTxns) CollectionName() string {
	return pendingTxnsColName
}

func (p *PendingTxns) EpochField() *string {
	return &pendingTxnsEpochField
}

func (p *PendingTxns) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(pendingTxnsEpochField, lower, upper), true
}

func (p *PendingTxns) Indexes() [][]string {
	return [][]string{
		[]string{"Addr"},
		[]string{pendingTxnsEpochField, "Addr"},
		[]string{pendingTxnsEpochField, "Addr", "Detail.TxnID"},
	}
}

func (p *PendingTxns) IsMutable() bool {
	return false
}
