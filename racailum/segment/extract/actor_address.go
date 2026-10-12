package extract

import (
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
)

// This file holds the incremental-write bookkeeping for the ActorAddress
// collection, in the same spirit as snapshot.go: no chain, no database, no
// network, so the writer (racailum/segment/extract/tipset) and the tests can
// exercise it in isolation.
//
// Background. ActorAddress is an insert-only collection (_id = ActorID,
// IsMutable() == false, model/actor_address.go) written from one tick per hour
// (ActorAddressTicks = 1, interval = 1h). The legacy extractor re-emitted every
// actor's row on every tick (~3.84e6 documents, of which only ~18/hour are new)
// and the driver silently swallowed the duplicate-key errors (InsertMany with
// SetOrdered(false), lib/mgoutil/docdb.go). Cost measured on mainnet: the
// traversal is ~14s, the insert is 425-456s (p50 435s, n=65, no fast sample),
// i.e. ~7m15s burnt every hour. Because a row's Epoch is frozen at the first
// insert, *not* re-sending an already-written ActorID is observationally
// identical to the legacy behaviour -- that equivalence is the whole change.
//
// Why the state is process-local (an in-memory set) and not a persistent
// cursor. `bell sethead` deletes ActorAddress as an *epoch* set: the collection
// is in epochCollections (cmd/bell/set_head.go:27) and setHead runs
// DeleteMany({"Epoch": {"$gt": head}}). Rows removed that way may carry any
// ActorID -- re-executing the rolled-back epochs re-creates actors under their
// original, small IDs. A "max(_id) watermark" cursor therefore loses those rows
// for good: the re-created ID sits below the watermark and is never re-emitted.
// A persisted *set* has the same staleness as the in-memory one (sethead does
// not know about it) but adds I/O, so it is strictly worse.
//
// The in-memory set has the opposite, benign failure mode: it is always wiped
// by a process restart, so the first tick after every restart is a full pass
// and re-emits every ActorID -- including any row a rollback removed. That is
// the healing contract: after a sethead (or any write that did not land), a
// restart makes the next tick self-correct. The cost is one full insert
// (~435s) per process start, deliberately logged as mode=full so it is
// distinguishable from an ordinary incremental tick.
//
// The write/persist HANDSHAKE (pending -> seen). "Emitted" is not the same as
// "written": the extractor emits the ActorAddress documents during extraction,
// but the persistence of those documents is *asynchronous* (Persist.Async, see
// Segment.persistPartDocs). Marking an ActorID as done the moment it is emitted
// therefore loses it for good whenever that part's insert fails or times out:
// the re-run of the same part would see the ID as already-seen, emit nothing,
// and the row would stay missing until the next process restart. So emission
// goes through a per-part *pending* set and only a confirmed write promotes it:
//
//   - ActorAddressPart.Filter (called from extractActorAddress) puts every ID
//     it emits into the owning part's pending list; the ID counts as "in
//     flight" for the whole process (so a concurrent sibling tipset of the
//     same part never re-emits it -- exactly-once within a part);
//   - the part's persistence is confirmed (Segment.persistPartDocs) => Commit
//     folds the part's pending IDs into `seen`;
//   - the part's persistence fails / times out, or the part is torn down before
//     it is persisted (extraction error, cancelled round, persist=false is not
//     a persisting part at all) => Discard drops the part's pending IDs, so the
//     next round re-emits them.
//
// A live in-memory state is still blind to `bell sethead` (see above) and is
// still wiped by a restart; the restart full pass remains the ultimate healer.

// ActorAddressMode is the emit strategy picked for one actor-address tick.
//
// The zero value is ActorAddressFull on purpose: a forgotten/blank state must
// degrade to the legacy "emit everything" behaviour, never to silently
// skipping rows.
type ActorAddressMode int

const (
	// ActorAddressFull: emit every ActorID (first tick after process start,
	// and the safe fallback for a nil/empty state).
	ActorAddressFull ActorAddressMode = iota
	// ActorAddressIncremental: emit only the ActorIDs not emitted before.
	ActorAddressIncremental
)

func (m ActorAddressMode) String() string {
	if m == ActorAddressIncremental {
		return "incremental"
	}

	return "full"
}

// ActorAddressWriteState remembers, for the lifetime of one process, which
// ActorIDs already had their ActorAddress row *confirmed written*. It is safe
// for concurrent use: several tipsets of one part are extracted in parallel and
// share the same state, and several parts of one batch are persisted in
// parallel too.
//
// `seen` holds the confirmed IDs; `pending` holds the IDs emitted but not yet
// confirmed, tagged per part in `parts` so a single part's outcome can be
// committed or dropped without disturbing another part still in flight.
type ActorAddressWriteState struct {
	mu      sync.Mutex
	seen    map[abi.ActorID]struct{}
	pending map[abi.ActorID]struct{}
	parts   map[*ActorAddressPart]struct{}
	ran     bool
}

// NewActorAddressWriteState returns an empty state. An empty state makes the
// first Filter call report ActorAddressFull, which is exactly the desired
// "first tick after a restart" behaviour.
func NewActorAddressWriteState() *ActorAddressWriteState {
	return &ActorAddressWriteState{
		seen:    make(map[abi.ActorID]struct{}),
		pending: make(map[abi.ActorID]struct{}),
		parts:   make(map[*ActorAddressPart]struct{}),
	}
}

// ensureLocked makes the zero value usable: a blank state degrades to "emit
// everything" instead of panicking on a nil map.
func (s *ActorAddressWriteState) ensureLocked() {
	if s.seen == nil {
		s.seen = make(map[abi.ActorID]struct{})
	}
	if s.pending == nil {
		s.pending = make(map[abi.ActorID]struct{})
	}
	if s.parts == nil {
		s.parts = make(map[*ActorAddressPart]struct{})
	}
}

// Plan reports the mode the next Filter call would use, without consuming
// anything. Full means "this process has not emitted anything yet".
func (s *ActorAddressWriteState) Plan() ActorAddressMode {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.ran {
		return ActorAddressFull
	}

	return ActorAddressIncremental
}

// BeginPart opens a new pending bucket for one part. The caller must finish it
// with Commit (the part's rows landed) or Discard (they did not), otherwise the
// IDs emitted under it stay pending forever and are never re-emitted.
func (s *ActorAddressWriteState) BeginPart() *ActorAddressPart {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureLocked()

	p := &ActorAddressPart{state: s}
	s.parts[p] = struct{}{}

	return p
}

// Seen reports how many distinct ActorIDs this process has confirmed written.
// It is what makes the in-memory cost observable in a log/metric.
func (s *ActorAddressWriteState) Seen() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.seen)
}

// Pending reports how many distinct ActorIDs are currently emitted but not yet
// confirmed written (handshake depth). It is what makes the handshake
// observable in a log/metric.
func (s *ActorAddressWriteState) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.pending)
}

// ActorAddressPart is one part's in-flight actor-address write: the ActorIDs
// that part emitted while extracting, held as *pending* (never counted as
// "seen") until the part's persistence is known to have landed.
//
// It is owned by exactly one part; its Filter is called by that part's several
// tipset goroutines concurrently.
type ActorAddressPart struct {
	state *ActorAddressWriteState
	// ids is appended under state.mu (from Filter) and read under state.mu
	// (from Pending / Commit / Discard), so the concurrent tipsets of one part
	// never race on it.
	ids []abi.ActorID
}

// Filter returns the subset of ids this part still has to write, records them
// as pending under this part and reports the mode this call ran in.
//
//   - mode == ActorAddressFull on the first Filter call of a process (every id
//     is emitted): this is where a rollback's deleted rows are re-written.
//   - mode == ActorAddressIncremental afterwards: only ids never seen (and not
//     already pending) are emitted, so the second call with the same batch
//     emits nothing.
//
// Filter is idempotent and concurrency-safe: an id is emitted by exactly one
// call, ever, within the lifetime of the state -- an id already pending in
// another part of the same batch is skipped too, so two parts of one batch
// never both carry the same row.
func (p *ActorAddressPart) Filter(ids []abi.ActorID) (emitted []abi.ActorID, mode ActorAddressMode) {
	s := p.state

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureLocked()

	mode = ActorAddressIncremental
	if !s.ran {
		mode = ActorAddressFull
	}
	s.ran = true

	emitted = make([]abi.ActorID, 0, len(ids))
	for _, id := range ids {
		if _, ok := s.seen[id]; ok {
			continue
		}
		if _, ok := s.pending[id]; ok {
			continue
		}
		s.pending[id] = struct{}{}
		p.ids = append(p.ids, id)
		emitted = append(emitted, id)
	}

	return emitted, mode
}

// Pending reports how many ActorIDs this part emitted but has not confirmed
// written yet (the handshake depth of this part).
func (p *ActorAddressPart) Pending() int {
	s := p.state

	s.mu.Lock()
	defer s.mu.Unlock()

	return len(p.ids)
}

// Commit folds this part's pending ActorIDs into the confirmed `seen` set. Call
// it exactly when the part's row insert is known to have landed.
//
// It is idempotent: a second call (or a Discard before/after it) is a no-op, so
// the success path can be guarded by a deferred Discard without double-counting.
func (p *ActorAddressPart) Commit() {
	s := p.state

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.parts[p]; !ok { // already finished
		return
	}

	for _, id := range p.ids {
		s.seen[id] = struct{}{}
		delete(s.pending, id)
	}
	delete(s.parts, p)
	p.ids = nil
}

// Discard drops this part's pending ActorIDs so the next round re-emits them.
// Call it whenever the part's rows are not confirmed written: an insert failure
// or timeout, an extraction error before persistence, a cancelled round.
//
// It is idempotent: a second call (or a Commit before/after it) is a no-op.
func (p *ActorAddressPart) Discard() {
	s := p.state

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.parts[p]; !ok { // already finished
		return
	}

	for _, id := range p.ids {
		delete(s.pending, id)
	}
	delete(s.parts, p)
	p.ids = nil
}
