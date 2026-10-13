package extract

import (
	"sync"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
)

// idsOf builds an ActorID slice, in order, for the table below.
func idsOf(vs ...uint64) []abi.ActorID {
	out := make([]abi.ActorID, 0, len(vs))
	for _, v := range vs {
		out = append(out, abi.ActorID(v))
	}

	return out
}

// actorAddressOutcome is what happened to a part *after* it was extracted: its
// rows either landed (commit), failed/timed out (discard), or are still in
// flight (leavePending -- only used to observe the pending state).
type actorAddressOutcome int

const (
	stepCommit actorAddressOutcome = iota
	stepDiscard
	stepLeavePending
)

// actorAddressStep is one part extraction against a state.
type actorAddressStep struct {
	name        string
	ids         []abi.ActorID
	outcome     actorAddressOutcome
	wantMode    ActorAddressMode
	wantEmitted []abi.ActorID
}

// runActorAddressSteps feeds each step to one state as a full part (BeginPart →
// Filter → commit/discard) and checks the mode and emitted set of every part.
func runActorAddressSteps(t *testing.T, s *ActorAddressWriteState, steps []actorAddressStep) {
	t.Helper()

	for _, step := range steps {
		part := s.BeginPart()
		emitted, mode := part.Filter(step.ids)
		if mode != step.wantMode {
			t.Fatalf("%s: mode = %s, want %s", step.name, mode, step.wantMode)
		}
		if !sameIDs(emitted, step.wantEmitted) {
			t.Fatalf("%s: emitted %v, want %v", step.name, emitted, step.wantEmitted)
		}

		switch step.outcome {
		case stepDiscard:
			part.Discard()
		case stepLeavePending:
			// deliberately neither commit nor discard
		default:
			part.Commit()
		}
	}
}

func sameIDs(a, b []abi.ActorID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// TestActorAddressWriteStateIncremental is the core contract: the first part
// emits everything (full), re-running the same batch after a successful write
// emits nothing (incremental), and only genuinely new ActorIDs are emitted
// afterwards.
func TestActorAddressWriteStateIncremental(t *testing.T) {
	cases := []struct {
		name  string
		steps []actorAddressStep
	}{
		{
			name: "same batch twice -> second emitted=0",
			steps: []actorAddressStep{
				{name: "part1 (first in process)", ids: idsOf(1, 2, 3), wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
				{name: "part2 (identical batch)", ids: idsOf(1, 2, 3), wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
			},
		},
		{
			name: "new ids -> only the new ones",
			steps: []actorAddressStep{
				{name: "part1", ids: idsOf(1, 2, 3), wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
				{name: "part2 (+4,5)", ids: idsOf(1, 2, 3, 4, 5), wantMode: ActorAddressIncremental, wantEmitted: idsOf(4, 5)},
				{name: "part3 (+6)", ids: idsOf(1, 2, 3, 4, 5, 6), wantMode: ActorAddressIncremental, wantEmitted: idsOf(6)},
				{name: "part4 (no new actor)", ids: idsOf(1, 2, 3, 4, 5, 6), wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
			},
		},
		{
			name: "ids are only emitted once even when the order changes",
			steps: []actorAddressStep{
				{name: "part1", ids: idsOf(7, 8), wantMode: ActorAddressFull, wantEmitted: idsOf(7, 8)},
				{name: "part2 (reordered, +9)", ids: idsOf(9, 7, 8), wantMode: ActorAddressIncremental, wantEmitted: idsOf(9)},
				{name: "part3 (reordered)", ids: idsOf(8, 9, 7), wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
			},
		},
		{
			name: "an empty first batch still reports full, and later ids are still emitted",
			steps: []actorAddressStep{
				{name: "part1 (empty chain window)", ids: idsOf(), wantMode: ActorAddressFull, wantEmitted: idsOf()},
				{name: "part2", ids: idsOf(1, 2), wantMode: ActorAddressIncremental, wantEmitted: idsOf(1, 2)},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runActorAddressSteps(t, NewActorAddressWriteState(), tc.steps)
		})
	}
}

// TestActorAddressPersistFailureReEmits is THE handshake contract: a part whose
// rows did NOT land (persist failure / timeout -> Discard) must re-emit on the
// next round, even though it was already "emitted" once. Without this, the
// ActorAddress rows of a failed async-persist part would be silently skipped
// until a process restart.
func TestActorAddressPersistFailureReEmits(t *testing.T) {
	s := NewActorAddressWriteState()

	// Round 1: the part emits everything, then its insert fails -> Discard.
	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "round1 part (persist fails)", ids: idsOf(1, 2, 3), outcome: stepDiscard, wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
	})
	if got := s.Seen(); got != 0 {
		t.Fatalf("failed persist must not count as written: Seen() = %d, want 0", got)
	}
	if got := s.Pending(); got != 0 {
		t.Fatalf("discarded part must not leave pending: Pending() = %d, want 0", got)
	}

	// Round 2: the same part is retried and now succeeds -> the rows are written.
	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "round2 part (persist lands)", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressIncremental, wantEmitted: idsOf(1, 2, 3)},
	})
	if got := s.Seen(); got != 3 {
		t.Fatalf("successful persist must count as written: Seen() = %d, want 3", got)
	}

	// Round 3: now that the rows landed, they are not sent again.
	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "round3 part (nothing new)", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
	})
}

// TestActorAddressPersistSuccessDoesNotReEmit pins the success side of the
// handshake: Commit promotes pending to seen, so the next round drops the ids.
func TestActorAddressPersistSuccessDoesNotReEmit(t *testing.T) {
	s := NewActorAddressWriteState()

	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "round1 part (persist lands)", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
	})
	if got := s.Pending(); got != 0 {
		t.Fatalf("committed part must clear pending: Pending() = %d, want 0", got)
	}

	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "round2 part (already written)", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
	})
}

// TestActorAddressPendingBlocksSiblingParts pins that an id already in flight
// (pending in another part of the same batch) is not re-emitted by a second
// part: two parts of one batch never both carry the same row.
func TestActorAddressPendingBlocksSiblingParts(t *testing.T) {
	s := NewActorAddressWriteState()

	p1 := s.BeginPart()
	e1, _ := p1.Filter(idsOf(1, 2, 3))
	if !sameIDs(e1, idsOf(1, 2, 3)) {
		t.Fatalf("part1 emitted %v, want 1,2,3", e1)
	}
	if got := s.Pending(); got != 3 {
		t.Fatalf("Pending() = %d, want 3 after part1 emission", got)
	}

	// Sibling part of the same batch, before part1's persist is confirmed.
	p2 := s.BeginPart()
	e2, _ := p2.Filter(idsOf(1, 2, 3))
	if len(e2) != 0 {
		t.Fatalf("part2 emitted %v, want none (ids are pending in part1)", e2)
	}

	// part1 fails: its ids go back to "emit me"; part2 (which emitted nothing)
	// is irrelevant.
	p1.Discard()
	if got := s.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want 0 after discard", got)
	}

	p3 := s.BeginPart()
	e3, _ := p3.Filter(idsOf(1, 2, 3))
	if !sameIDs(e3, idsOf(1, 2, 3)) {
		t.Fatalf("after a failed sibling, re-run must emit all ids, got %v", e3)
	}
}

// TestActorAddressWriteStateZeroValueIsFull pins the safety property that a
// blank (nil-able / forgotten) state degrades to the legacy "emit everything"
// behaviour instead of silently skipping rows.
func TestActorAddressWriteStateZeroValueIsFull(t *testing.T) {
	if got := ActorAddressMode(0); got != ActorAddressFull {
		t.Fatalf("zero ActorAddressMode = %s, want full", got)
	}

	var zero ActorAddressWriteState
	part := zero.BeginPart()
	emitted, mode := part.Filter(idsOf(1, 2))
	if mode != ActorAddressFull {
		t.Fatalf("mode = %s, want full", mode)
	}
	if len(emitted) != 2 {
		t.Fatalf("emitted = %v, want both ids", emitted)
	}
}

// TestActorAddressWriteStateRestartIsFullPass pins the restart contract: the
// state is process-local, so a new process (a fresh state) turns its first
// part back into a full pass that re-emits every ActorID.
func TestActorAddressWriteStateRestartIsFullPass(t *testing.T) {
	before := NewActorAddressWriteState()
	runActorAddressSteps(t, before, []actorAddressStep{
		{name: "part1", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
		{name: "part2", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
	})

	if got := before.Seen(); got != 3 {
		t.Fatalf("Seen() = %d, want 3", got)
	}

	// "restart": the state does not survive it.
	after := NewActorAddressWriteState()
	if got := after.Plan(); got != ActorAddressFull {
		t.Fatalf("fresh state Plan() = %s, want full", got)
	}
	runActorAddressSteps(t, after, []actorAddressStep{
		{name: "part1 of the new process", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
	})
}

// TestActorAddressRollbackSetheadRequiresRestart pins the semantic chosen for
// the `bell sethead` case (cmd/bell/set_head.go:27): ActorAddress is deleted
// there as an *epoch* set, and the in-process state cannot observe it. So a
// live process keeps skipping the rows a rollback removed (documented gap) and
// only a RESTART re-emits them -- which is precisely why the state is
// process-local rather than a max(_id) cursor that could never recover them.
func TestActorAddressRollbackSetheadRequiresRestart(t *testing.T) {
	s := NewActorAddressWriteState()
	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "part before the rollback", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
	})

	// sethead to a lower head deletes the ActorAddress rows inserted above it
	// (DeleteMany {"Epoch": {"$gt": head}}) -- here, the rows for 2 and 3. The
	// process is NOT restarted and its state does not know about the delete.
	runActorAddressSteps(t, s, []actorAddressStep{
		{name: "part after sethead, no restart (deleted rows stay skipped: known gap)", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressIncremental, wantEmitted: idsOf()},
	})

	// The healing path: restart. The empty state makes the first part a full
	// pass, which re-writes every row the rollback deleted.
	restarted := NewActorAddressWriteState()
	runActorAddressSteps(t, restarted, []actorAddressStep{
		{name: "first part after restart (every row rewritten)", ids: idsOf(1, 2, 3), outcome: stepCommit, wantMode: ActorAddressFull, wantEmitted: idsOf(1, 2, 3)},
	})
}

// TestActorAddressWriteStatePlan checks Plan is a pure peek: it reports full
// until the first Filter call and never consumes the ids.
func TestActorAddressWriteStatePlan(t *testing.T) {
	s := NewActorAddressWriteState()
	if got := s.Plan(); got != ActorAddressFull {
		t.Fatalf("Plan before any part = %s, want full", got)
	}
	if got := s.Seen(); got != 0 {
		t.Fatalf("Seen before any part = %d, want 0", got)
	}

	s.BeginPart().Filter(idsOf(1))

	if got := s.Plan(); got != ActorAddressIncremental {
		t.Fatalf("Plan after a part = %s, want incremental", got)
	}
	if got := s.Seen(); got != 0 {
		t.Fatalf("Seen after an uncommitted part = %d, want 0 (not written yet)", got)
	}
	if got := s.Pending(); got != 1 {
		t.Fatalf("Pending after an uncommitted part = %d, want 1", got)
	}
}

// TestActorAddressWriteStateConcurrentExactlyOnce is the concurrency contract:
// the several tipsets of ONE part share one part bucket and race through
// Filter, yet each ActorID is emitted by exactly one of them and the part
// carries it exactly once.
func TestActorAddressWriteStateConcurrentExactlyOnce(t *testing.T) {
	s := NewActorAddressWriteState()
	part := s.BeginPart()

	const workers = 8
	batch := idsOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	var (
		mu       sync.Mutex
		perID    = map[abi.ActorID]int{}
		fullRuns int
		wg       sync.WaitGroup
	)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			emitted, mode := part.Filter(batch)

			mu.Lock()
			defer mu.Unlock()
			if mode == ActorAddressFull {
				fullRuns++
			}
			for _, id := range emitted {
				perID[id]++
			}
		}()
	}
	wg.Wait()

	if fullRuns != 1 {
		t.Fatalf("full runs = %d, want exactly 1 (the first part of the process)", fullRuns)
	}
	if len(perID) != len(batch) {
		t.Fatalf("distinct ids emitted = %d, want %d", len(perID), len(batch))
	}
	for id, n := range perID {
		if n != 1 {
			t.Fatalf("id %d emitted %d times, want exactly 1", id, n)
		}
	}
	if got := part.Pending(); got != len(batch) {
		t.Fatalf("part Pending() = %d, want %d", got, len(batch))
	}

	part.Commit()
	if got := s.Seen(); got != len(batch) {
		t.Fatalf("Seen() = %d, want %d", got, len(batch))
	}
	if got := s.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want 0 after commit", got)
	}
}

func TestActorAddressModeString(t *testing.T) {
	cases := []struct {
		mode ActorAddressMode
		want string
	}{
		{mode: ActorAddressFull, want: "full"},
		{mode: ActorAddressIncremental, want: "incremental"},
		{mode: ActorAddressMode(42), want: "full"}, // unknown must not print empty
	}

	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Fatalf("mode %d String() = %q, want %q", int(tc.mode), got, tc.want)
		}
	}
}

// TestActorAddressPartCommitDiscardIdempotent pins that the two finishing calls
// can never double-count: whichever runs first wins, the next is a no-op.
func TestActorAddressPartCommitDiscardIdempotent(t *testing.T) {
	s := NewActorAddressWriteState()

	committed := s.BeginPart()
	committed.Filter(idsOf(1, 2))
	committed.Commit()
	committed.Discard() // must not undo the commit
	if got := s.Seen(); got != 2 {
		t.Fatalf("Seen() = %d, want 2 (discard after commit is a no-op)", got)
	}

	discarded := s.BeginPart()
	discarded.Filter(idsOf(3, 4))
	discarded.Discard()
	discarded.Commit() // must not resurrect a discarded part
	if got := s.Seen(); got != 2 {
		t.Fatalf("Seen() = %d, want 2 (commit after discard is a no-op)", got)
	}
	if got := s.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want 0", got)
	}
}
