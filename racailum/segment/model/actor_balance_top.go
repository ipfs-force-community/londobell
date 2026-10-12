package model

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

var (
	_ common.IndexedDocument = (*ActorBalanceTop)(nil)

	actorBalanceTopColName    = getColName(ActorBalanceTop{})
	actorBalanceTopEpochField = extractEpochFieldName(ActorBalanceTop{})
)

// ActorBalanceTopCollection is the mongo collection that holds the daily
// top-N richlist snapshot. Readers must reference this constant instead of a
// literal so a rename cannot silently point them at an empty collection.
const ActorBalanceTopCollection = "ActorBalanceTop"

func init() {
	schema.Register(
		schema.Model{
			Name: "actor-balance-top",
			D:    &ActorBalanceTop{},
		},
	)
}

// ActorBalanceTop is one row of the daily top-N richlist snapshot.
//
// Background (2026-10-12): ActorBalance no longer carries any epoch with the
// whole actor set — the zero hour now writes only the actors whose Balance or
// Code changed, exactly like a regular tick. The richlist needs a whole set at
// one epoch, so the writer keeps the N largest balances it sees while it is
// already walking the state tree for that epoch and writes them here.
//
// Unlike ActorBalance (1.706e9 rows / 589GB) this collection is tiny: N rows
// per day (N defaults to 10000) ≈ a few MB per year, and it exists for
// exactly one consumer — the richlist endpoint.
//
// Epoch is the snapshot epoch so a reader can take "the latest row-set at or
// below X"; Rank is 1-based (1 = largest balance) and is what makes the
// richlist's ordering exact and stable: equal balances are broken by address
// ascending, so the same chain state always yields the same list.
type ActorBalanceTop struct {
	ActorStateExBasic `bson:",inline"`

	Balance abi.TokenAmount
	Code    string
	Rank    int
}

// CollectionName impl common.Document
func (a *ActorBalanceTop) CollectionName() string {
	return actorBalanceTopColName
}

// EpochField impl common.Document
func (a *ActorBalanceTop) EpochField() *string {
	return &actorBalanceTopEpochField
}

// ResetPolicy impl common.Document
func (a *ActorBalanceTop) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(actorBalanceTopEpochField, lower, upper), true
}

func (a *ActorBalanceTop) Indexes() [][]string {
	return [][]string{
		{"Epoch", "Balance"},
		{"Epoch", "Addr"},
	}
}

func (a *ActorBalanceTop) IsMutable() bool {
	return false
}
