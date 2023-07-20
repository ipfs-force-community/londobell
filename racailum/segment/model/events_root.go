package model

import (
	"encoding/json"

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
	Events []byte
	Epoch  abi.ChainEpoch
}

func (f *EventsRoot) Indexes() [][]string {
	return [][]string{}
}

func NewEventsRoot(root cid.Cid, events []types.Event, epoch abi.ChainEpoch) (*EventsRoot, error) {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}

	return &EventsRoot{
		Root:   root,
		Events: eventsJSON,
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

func (f *EventsRoot) IsMutable() bool {
	return false
}
