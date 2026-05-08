package model

import (
	"strconv"
	"strings"

	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                    common.IndexedDocument = (*ActorEvent)(nil)
	actorEventEpochField                        = extractEpochFieldName(ActorEvent{})
	actorEventColName                           = getColName(ActorEvent{})
)

// ActorMessage records messages for actor
type ActorEvent struct {
	ID        string `mir:"-" bson:"_id"`
	ActorID   address.Address
	Epoch     abi.ChainEpoch `mir:"-"`
	Cid       cid.Cid
	SignedCid cid.Cid
	Topics    []ethtypes.EthHash
	Data      ethtypes.EthBytes
	LogIndex  uint64
	Removed   bool
}

func NewActorEvent(actorID address.Address, epoch abi.ChainEpoch, cid, signedCid cid.Cid, topics []ethtypes.EthHash, data ethtypes.EthBytes, logIndex uint64, removed bool, seq []int) (*ActorEvent, error) {
	ae := &ActorEvent{
		ActorID:   actorID,
		Epoch:     epoch,
		Cid:       cid,
		SignedCid: signedCid,
		Topics:    topics,
		Data:      data,
		LogIndex:  logIndex,
		Removed:   removed,
	}

	ae.genID(epoch, logIndex, seq)
	return ae, nil
}

// Indexes impl common.Indexed
func (ae *ActorEvent) Indexes() [][]string {
	return [][]string{
		[]string{"ActorID", actorEventEpochField, "LogIndex"},
	}
}

// CollectionName impl common.Document
func (ae *ActorEvent) CollectionName() string {
	return actorEventColName
}

// EpochField impl common.Document
func (ae *ActorEvent) EpochField() *string {
	return &actorEventEpochField
}

// ResetPolicy impl common.Document
func (ae *ActorEvent) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(actorEventEpochField, lower, upper), true
}

func (ae *ActorEvent) genID(epoch abi.ChainEpoch, logIndex uint64, seq []int) {
	n := 20 + len(seq)*6
	var b strings.Builder
	b.Grow(n)
	b.WriteString(strconv.FormatInt(int64(epoch), 10))
	b.WriteByte('-')
	for i := range seq {
		if i > 0 {
			b.WriteByte('-')
		}
		s := strconv.Itoa(seq[i])
		for j := 5 - len(s); j > 0; j-- {
			b.WriteByte('0')
		}
		b.WriteString(s)
	}
	b.WriteByte('-')
	b.WriteString(strconv.FormatUint(logIndex, 10))
	ae.ID = b.String()
}

func (ae *ActorEvent) IsMutable() bool {
	return false
}
