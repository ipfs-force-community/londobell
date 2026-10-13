package segment

import (
	"testing"

	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

// TestAllToCheckTableListCoversActorBalanceTop pins that the segment.check tool
// verifies the new daily top-N collection.
//
// check (cmd/bell/check.go) iterates Options.AllToCheckTableList and fails a
// table that has no data at its expected epoch; a collection missing from the
// list is never checked at all. ActorBalanceTop is the richlist's only source
// (ActorBalance no longer has any epoch with the whole actor set), so a stalled
// or short ActorBalanceTop would silently degrade the richlist -- the check
// must cover it. model.ActorBalanceTopCollection keeps the name single-sourced.
func TestAllToCheckTableListCoversActorBalanceTop(t *testing.T) {
	want := model.ActorBalanceTopCollection
	opt := DefaultOptions()

	for _, tbl := range opt.AllToCheckTableList {
		if tbl == want {
			return
		}
	}

	t.Fatalf("AllToCheckTableList must contain %q so check verifies the richlist source; got %v", want, opt.AllToCheckTableList)
}
