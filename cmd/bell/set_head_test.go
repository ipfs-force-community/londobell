package main

import (
	"testing"

	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

// TestEpochCollectionsCoversActorBalanceTop pins the sethead rollback contract.
//
// setHead cleans every collection in epochCollections with
// DeleteMany({"Epoch": {"$gt": head}}). A collection missing from that list
// keeps its rows above the rolled-back water level, so rolling the head back
// below a local midnight would leave the daily top-N richlist snapshot in
// place: the richlist would then serve entries that are newer than the head,
// with no way to tell. model.ActorBalanceTopCollection is the single source of
// the collection name, so this compares against it instead of a literal.
func TestEpochCollectionsCoversActorBalanceTop(t *testing.T) {
	want := model.ActorBalanceTopCollection

	found := false
	for _, c := range epochCollections {
		if c == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("epochCollections must contain %q (sethead rollback would leave its rows above the water level): %v", want, epochCollections)
	}

	// The top-N collection is epoch-keyed: it must be swept by the epoch path,
	// not by the _id or deal-proposal ones.
	for _, c := range idCollections {
		if c == want {
			t.Fatalf("%q is epoch-keyed and must not be in idCollections", want)
		}
	}
	for _, c := range dealProposalCollections {
		if c == want {
			t.Fatalf("%q must not be in dealProposalCollections", want)
		}
	}
}
