package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                 common.IndexedDocument = (*EthHash)(nil)
	ethHashColName                           = getColName(EthHash{})
	ethHashEpochField                        = extractEpochFieldName(EthHash{})
)

// EthHash records map for TxHash to Cid
type EthHash struct {
	TxHash ethtypes.EthHash `mir:"-" bson:"_id"`
	Cid    cid.Cid
	Epoch  abi.ChainEpoch
}

func NewEthHash(txHash ethtypes.EthHash, cid cid.Cid, epoch abi.ChainEpoch) (*EthHash, error) {
	return &EthHash{
		TxHash: txHash,
		Cid:    cid,
		Epoch:  epoch,
	}, nil
}

// Indexes impl common.Indexed
func (eh *EthHash) Indexes() [][]string {
	return [][]string{
		[]string{"Cid", ethHashEpochField},
	}
}

// CollectionName impl common.Document
func (eh *EthHash) CollectionName() string {
	return ethHashColName
}

// EpochField impl common.Document
func (eh *EthHash) EpochField() *string {
	return &ethHashEpochField
}

// ResetPolicy impl common.Document
func (eh *EthHash) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(ethHashEpochField, lower, upper), true
}

func (eh *EthHash) IsMutable() bool {
	return false
}
