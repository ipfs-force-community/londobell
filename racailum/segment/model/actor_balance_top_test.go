package model

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
)

// TestActorBalanceTopDocument pins the contract the writer and the readers
// rely on: the collection name (readers look it up by string), the epoch field
// (readers take "the latest row-set at or below X"), and insert-only semantics.
func TestActorBalanceTopDocument(t *testing.T) {
	doc := &ActorBalanceTop{}

	if got := doc.CollectionName(); got != ActorBalanceTopCollection {
		t.Fatalf("CollectionName = %q, want the pinned constant %q", got, ActorBalanceTopCollection)
	}
	if got := (&ActorBalance{}).CollectionName(); got == doc.CollectionName() {
		t.Fatalf("top-N collection must differ from ActorBalance, both are %q", got)
	}

	if got := doc.EpochField(); got == nil || *got != "Epoch" {
		t.Fatalf("EpochField = %v, want \"Epoch\"", got)
	}

	if doc.IsMutable() {
		t.Fatal("top-N rows are insert-only snapshots; IsMutable must be false")
	}

	filter, ok := doc.ResetPolicy(nil, nil)
	if !ok {
		t.Fatal("ResetPolicy must handle epoch ranges (sethead rollback deletes by epoch)")
	}
	if filter == nil {
		t.Fatal("ResetPolicy returned a nil filter")
	}

	var _ abi.ChainEpoch = doc.Epoch

	idx := doc.Indexes()
	if len(idx) == 0 {
		t.Fatal("top-N rows need an Epoch-leading index for the daily read")
	}
	for _, ix := range idx {
		if len(ix) == 0 || ix[0] != "Epoch" {
			t.Fatalf("every index must lead with Epoch, got %v", ix)
		}
	}
}
