package extract

import (
	"container/heap"
	"sort"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
)

// DefaultActorBalanceTopN is the size of the daily richlist snapshot when the
// config leaves it unset (0). The richlist endpoint returns 1000 rows, so this
// keeps a 10x margin: enough headroom that the read-time code/positive-balance
// filter never runs the ranking dry.
const DefaultActorBalanceTopN = 10000

// RichListCodes are the actor codes the richlist ranking is restricted to
// (the short names ActorBalance/ ActorBalanceTop store in Code). They are the
// same three the upstream richlist.js has always matched.
var RichListCodes = []string{"account", "multisig", "storageminer"}

// TopBalanceCandidate is one actor the daily snapshot algorithm considers: its
// ID address, current balance, actor code and head (the head is only needed to
// mint the row's _id at emission time).
type TopBalanceCandidate struct {
	Addr    address.Address
	Balance abi.TokenAmount
	Code    string
	Head    cid.Cid
}

// ranksBefore is the richlist's total order: larger balance first, and equal
// balances broken by address ascending. Addresses are unique per actor, so this
// is a strict order — the output of the selection below cannot depend on the
// order actors were visited in.
func (c TopBalanceCandidate) ranksBefore(o TopBalanceCandidate) bool {
	if cmp := big.Cmp(c.Balance, o.Balance); cmp != 0 {
		return cmp > 0
	}

	return strings.Compare(c.Addr.String(), o.Addr.String()) < 0
}

// topBalancesHeap is a min-heap under ranksBefore: its root is the *worst*
// candidate currently retained, which is the one to evict once N are held.
type topBalancesHeap []TopBalanceCandidate

func (h topBalancesHeap) Len() int { return len(h) }
func (h topBalancesHeap) Less(i, j int) bool {
	return h[j].ranksBefore(h[i])
}
func (h topBalancesHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *topBalancesHeap) Push(x interface{}) {
	*h = append(*h, x.(TopBalanceCandidate))
}
func (h *topBalancesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]

	return x
}

// TopBalances keeps the N greatest candidates seen so far under ranksBefore.
//
// It is the "fixed-size min-heap" half of the daily richlist snapshot: the
// writer feeds it every actor it walks past (the same traversal that produces
// the ActorBalance rows — no second pass over the state tree), and reads back
// an exactly ordered list. Nothing here samples or approximates: an actor is
// only ever dropped when N strictly better ones are already held.
//
// N <= 0 disables the selection (Add is a no-op, Sorted is empty).
type TopBalances struct {
	n     int
	items topBalancesHeap
}

// NewTopBalances returns a selection keeping the n greatest candidates.
func NewTopBalances(n int) *TopBalances {
	if n <= 0 {
		return &TopBalances{}
	}

	return &TopBalances{n: n, items: make(topBalancesHeap, 0, n)}
}

// Len is the number of candidates currently retained (<= N).
func (t *TopBalances) Len() int { return len(t.items) }

// Add offers one candidate. The cheap comparison against the root skips the
// heap push for the overwhelming majority of actors (whose balance is far
// below the current Nth), which is what makes running this inside the full
// state-tree walk affordable.
func (t *TopBalances) Add(c TopBalanceCandidate) {
	if t.n <= 0 {
		return
	}

	if len(t.items) < t.n {
		heap.Push(&t.items, c)

		return
	}

	// root is the worst retained candidate: keep this one only if it ranks
	// before that root.
	if !c.ranksBefore(t.items[0]) {
		return
	}

	heap.Push(&t.items, c)
	heap.Pop(&t.items)
}

// WouldKeep reports whether a candidate with this key could be retained right
// now: the heap is not full yet, or the key outranks the current worst.
//
// It exists so a caller holding an expensive-to-build candidate (an actor name
// costs a map lookup, an Sprintf and a slice allocation) can skip building it
// for the overwhelming majority of actors, instead of paying that for all
// ~3.84e6 of them on the daily walk. It is a pure pre-check, not a semantic
// shortcut: Add applies the same comparison again, so skipping the
// would-not-keep candidates cannot change the retained set (pinned by
// TestTopBalancesWouldKeepMatchesAdd).
func (t *TopBalances) WouldKeep(addr address.Address, balance abi.TokenAmount) bool {
	if t.n <= 0 {
		return false
	}

	if len(t.items) < t.n {
		return true
	}

	return TopBalanceCandidate{Addr: addr, Balance: balance}.ranksBefore(t.items[0])
}

// Sorted returns the retained candidates in richlist order (balance desc,
// address asc). It never mutates the selection.
func (t *TopBalances) Sorted() []TopBalanceCandidate {
	out := make([]TopBalanceCandidate, len(t.items))
	copy(out, t.items)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ranksBefore(out[j])
	})

	return out
}

// TopK returns at most the k best candidates in richlist order. k < 0 means
// "all retained".
func (t *TopBalances) TopK(k int) []TopBalanceCandidate {
	out := t.Sorted()
	if k >= 0 && k < len(out) {
		out = out[:k]
	}

	return out
}

// RichListRows reproduces the richlist read over a set of rows that already
// went through the top-N selection: keep the richlist codes with a positive
// balance, order by balance desc then address asc, and take at most k rows
// (k < 0 keeps all).
//
// It is the pure twin of the mongo pipeline the endpoint actually runs, so a
// test can pin "the response built from the top-N snapshot equals the response
// built from a full sort of every candidate" without a database.
func RichListRows(rows []TopBalanceCandidate, codes []string, k int) []TopBalanceCandidate {
	allowed := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		allowed[c] = struct{}{}
	}

	out := make([]TopBalanceCandidate, 0, len(rows))
	for i := range rows {
		if _, ok := allowed[rows[i].Code]; !ok {
			continue
		}

		if big.Cmp(rows[i].Balance, big.Zero()) <= 0 {
			continue
		}

		out = append(out, rows[i])
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ranksBefore(out[j])
	})

	if k >= 0 && k < len(out) {
		out = out[:k]
	}

	return out
}
