package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*EventsRoot)(nil)

	eventsRootColName    = getColName(EventsRoot{})
	eventsRootEpochField = extractEpochFieldName(EventsRoot{})
)

type EventsRoot struct {
	Root   cid.Cid `mir:"-" bson:"_id"`
	Events []types.Event
	Epoch  abi.ChainEpoch
}

func (f *EventsRoot) Indexes() [][]string {
	return [][]string{}
}

func NewEventsRoot(root cid.Cid, events []types.Event, epoch abi.ChainEpoch) (*EventsRoot, error) {
	return &EventsRoot{
		Root:   root,
		Events: events,
		Epoch:  epoch,
	}, nil
}

// CollectionName impl common.Document
func (f *EventsRoot) CollectionName() string {
	return eventsRootColName
}

// EpochField impl common.Document
func (f *EventsRoot) EpochField() *string {
	return &eventsRootEpochField
}

// ResetPolicy impl common.Document
func (f *EventsRoot) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(eventsRootEpochField, lower, upper), true
}
