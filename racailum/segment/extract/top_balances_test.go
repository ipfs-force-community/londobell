package extract

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
)

// A spare code ("evm") is deliberately in the mix: the top-N selection ranks
// *every* actor and the richlist filter is applied at read time, so the
// selection must not care about codes.
var topTestCodes = []string{"account", "multisig", "storageminer", "evm"}

const richListReturned = 1000

func topTestCandidate(t *testing.T, id int, balance int64, code string) TopBalanceCandidate {
	t.Helper()

	addr, err := address.NewIDAddress(uint64(id) + 1000)
	if err != nil {
		t.Fatalf("new id address %d: %v", id, err)
	}

	return TopBalanceCandidate{Addr: addr, Balance: big.NewInt(balance), Code: code}
}

// collect feeds every candidate into the selection in order and returns the
// sorted result — the writer's exact code path.
func collect(cands []TopBalanceCandidate, n int) []TopBalanceCandidate {
	top := NewTopBalances(n)
	for i := range cands {
		top.Add(cands[i])
	}

	return top.Sorted()
}

// naiveTopN is the reference: sort everything, take N. The selection must be
// exactly this, never an approximation.
func naiveTopN(cands []TopBalanceCandidate, n int) []TopBalanceCandidate {
	out := make([]TopBalanceCandidate, len(cands))
	copy(out, cands)
	sort.Slice(out, func(i, j int) bool { return out[i].ranksBefore(out[j]) })

	if n >= 0 && n < len(out) {
		out = out[:n]
	}

	return out
}

func assertSameRows(t *testing.T, got, want []TopBalanceCandidate) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("row count = %d, want %d\n got: %v\nwant: %v", len(got), len(want), got, want)
	}

	for i := range got {
		if got[i].Addr != want[i].Addr {
			t.Fatalf("row %d: addr = %s, want %s", i, got[i].Addr, want[i].Addr)
		}
		if big.Cmp(got[i].Balance, want[i].Balance) != 0 {
			t.Fatalf("row %d: balance = %s, want %s", i, got[i].Balance, want[i].Balance)
		}
		if got[i].Code != want[i].Code {
			t.Fatalf("row %d: code = %s, want %s", i, got[i].Code, want[i].Code)
		}
	}
}

// TestTopBalancesMatchesFullSort is the precision requirement: whatever the
// candidate count, the heap must produce exactly the row list a full sort would
// (1). The generator deliberately repeats balances so ties are exercised.
func TestTopBalancesMatchesFullSort(t *testing.T) {
	rng := rand.New(rand.NewSource(20261012))

	for _, n := range []int{1, 2, 10, 100, 1000} {
		for _, m := range []int{0, 1, n - 1, n, n + 1, 3 * n, 7 * n} {
			if m < 0 {
				continue
			}

			cands := make([]TopBalanceCandidate, 0, m)
			for i := 0; i < m; i++ {
				// Heavy ties: only a handful of distinct balances.
				bal := int64(rng.Intn(6)) * 1000
				cands = append(cands, topTestCandidate(t, i, bal, topTestCodes[rng.Intn(len(topTestCodes))]))
			}

			assertSameRows(t, collect(cands, n), naiveTopN(cands, n))
		}
	}
}

// TestTopBalancesBoundaries covers the explicit edges: fewer candidates than N,
// exactly N, one short/over, and the all-equal case where the address tiebreak
// is what decides.
func TestTopBalancesBoundaries(t *testing.T) {
	const n = 4

	cases := []struct {
		name     string
		balances []int64
		wantLen  int
	}{
		{name: "empty", balances: nil, wantLen: 0},
		{name: "one candidate", balances: []int64{50}, wantLen: 1},
		{name: "fewer than N", balances: []int64{50, 10, 20}, wantLen: 3},
		{name: "exactly N", balances: []int64{50, 10, 20, 30}, wantLen: n},
		{name: "N+1 candidates", balances: []int64{50, 10, 20, 30, 40}, wantLen: n},
		{name: "all equal balances", balances: []int64{7, 7, 7, 7, 7, 7}, wantLen: n},
		{name: "all zero balances", balances: []int64{0, 0, 0, 0, 0}, wantLen: n},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cands := make([]TopBalanceCandidate, 0, len(tc.balances))
			for i, b := range tc.balances {
				cands = append(cands, topTestCandidate(t, i, b, "account"))
			}

			got := collect(cands, n)
			assertSameRows(t, got, naiveTopN(cands, n))

			if len(got) != tc.wantLen {
				t.Fatalf("kept %d rows, want %d", len(got), tc.wantLen)
			}

			for i := 1; i < len(got); i++ {
				if got[i-1].ranksBefore(got[i]) == false {
					t.Fatalf("row %d does not rank before row %d: %v / %v", i-1, i, got[i-1], got[i])
				}
			}
		})
	}

	// The all-equal case must be the *lowest* four addresses, by ascending
	// address — the stable tiebreak, not insertion order.
	equal := make([]TopBalanceCandidate, 0, 6)
	for i := 0; i < 6; i++ {
		equal = append(equal, topTestCandidate(t, i, 7, "account"))
	}

	got := collect(equal, n)
	for i := 1; i < len(got); i++ {
		if got[i-1].Addr.String() >= got[i].Addr.String() {
			t.Fatalf("equal balances must be ordered by address ascending: %s >= %s", got[i-1].Addr, got[i].Addr)
		}
	}
	if got[0].Addr.String() != equal[0].Addr.String() {
		t.Fatalf("tiebreak lost the lowest address: got %s, want %s", got[0].Addr, equal[0].Addr)
	}
}

// TestTopBalancesOrderIndependent is requirement (3): the retained set and its
// order must not depend on the order actors were visited in.
func TestTopBalancesOrderIndependent(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	base := make([]TopBalanceCandidate, 0, 5000)
	for i := 0; i < 5000; i++ {
		// Values repeat so both the balance and the address tiebreak are used.
		base = append(base, topTestCandidate(t, i, int64(rng.Intn(20))*500, topTestCodes[i%len(topTestCodes)]))
	}

	const n = 137
	want := naiveTopN(base, n)

	shuffled := make([]TopBalanceCandidate, len(base))
	for attempt := 0; attempt < 20; attempt++ {
		copy(shuffled, base)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

		assertSameRows(t, collect(shuffled, n), want)
	}

	// Reversed order is the adversarial case for a heap.
	reversed := make([]TopBalanceCandidate, len(base))
	for i := range base {
		reversed[i] = base[len(base)-1-i]
	}
	assertSameRows(t, collect(reversed, n), want)
}

// TestTopBalancesWouldKeepMatchesAdd pins the cheap pre-check as semantics
// preserving: skipping every candidate WouldKeep rejects must leave exactly the
// same retained set as feeding them all in.
func TestTopBalancesWouldKeepMatchesAdd(t *testing.T) {
	rng := rand.New(rand.NewSource(31337))

	for _, n := range []int{1, 7, 64, 500} {
		cands := make([]TopBalanceCandidate, 0, 5000)
		for i := 0; i < 5000; i++ {
			cands = append(cands, topTestCandidate(t, i, int64(rng.Intn(12))*250, topTestCodes[i%len(topTestCodes)]))
		}

		all := NewTopBalances(n)
		guarded := NewTopBalances(n)
		skipped := 0
		for i := range cands {
			all.Add(cands[i])
			if !guarded.WouldKeep(cands[i].Addr, cands[i].Balance) {
				skipped++
				continue
			}
			guarded.Add(cands[i])
		}

		assertSameRows(t, guarded.Sorted(), all.Sorted())
		if skipped == 0 {
			t.Fatalf("n=%d: the pre-check never fired, the test is not exercising anything", n)
		}
	}
}

// TestTopBalancesZeroDisables pins the off switch: N <= 0 keeps nothing, so a
// misconfigured writer emits no top-N rows instead of all of them.
func TestTopBalancesZeroDisables(t *testing.T) {
	for _, n := range []int{0, -1} {
		top := NewTopBalances(n)
		top.Add(topTestCandidate(t, 1, 100, "account"))
		if top.Len() != 0 {
			t.Fatalf("n=%d: kept %d rows, want 0", n, top.Len())
		}
		if len(top.Sorted()) != 0 {
			t.Fatalf("n=%d: Sorted() must be empty", n)
		}
	}
}

// TestTopBalancesTopK pins the reader-side slice used by the consistency test.
func TestTopBalancesTopK(t *testing.T) {
	cands := make([]TopBalanceCandidate, 0, 10)
	for i := 0; i < 10; i++ {
		cands = append(cands, topTestCandidate(t, i, int64(i), "account"))
	}

	top := NewTopBalances(10)
	for i := range cands {
		top.Add(cands[i])
	}

	if got := top.TopK(3); len(got) != 3 || got[0].Balance.Int64() != 9 || got[2].Balance.Int64() != 7 {
		t.Fatalf("TopK(3) = %v, want the three largest", got)
	}
	if got := top.TopK(-1); len(got) != 10 {
		t.Fatalf("TopK(-1) = %d rows, want all 10", len(got))
	}
}

// TestRichListResponseEqualsFullSort is the headline requirement: the list
// served from the daily top-N snapshot must be exactly the list a full sort of
// every candidate would produce. N here is the production default, so this also
// exercises the 10x margin over the 1000 rows the endpoint returns.
func TestRichListResponseEqualsFullSort(t *testing.T) {
	rng := rand.New(rand.NewSource(4242))

	// 9000 non-richlist actors sit above the richest 1000 richlist candidates:
	// the deliberately worst case the 10x margin has to absorb.
	cands := make([]TopBalanceCandidate, 0, 40000)
	for i := 0; i < 9000; i++ {
		cands = append(cands, topTestCandidate(t, i, 2_000_000-int64(i), "evm"))
	}
	for i := 0; i < 1000; i++ {
		code := RichListCodes[i%len(RichListCodes)]
		cands = append(cands, topTestCandidate(t, 9000+i, 1_000_000-int64(i), code))
	}
	for i := 0; i < 30000; i++ {
		// Small, heavily tied filler with some zero balances.
		cands = append(cands, topTestCandidate(t, 10000+i, int64(rng.Intn(4))*100, topTestCodes[rng.Intn(len(topTestCodes))]))
	}

	top := NewTopBalances(EffectiveActorBalanceTopN(0))
	for i := range cands {
		top.Add(cands[i])
	}

	fromSnapshot := RichListRows(top.Sorted(), RichListCodes, richListReturned)
	fromFullSort := RichListRows(cands, RichListCodes, richListReturned)

	assertSameRows(t, fromSnapshot, fromFullSort)

	if len(fromFullSort) != richListReturned {
		t.Fatalf("expected a full %d-row list, got %d", richListReturned, len(fromFullSort))
	}
}

// TestRichListResponseNeedsTopNMargin documents the one assumption the design
// rests on. When more than N actors outrank the last richlist row, the top-N
// cut drops richlist rows and the response shrinks. This is exactly why N is
// 10x the returned 1000; if the chain's actor mix ever changes, N must grow.
func TestRichListResponseNeedsTopNMargin(t *testing.T) {
	const n = 100

	cands := make([]TopBalanceCandidate, 0, n+1000)
	for i := 0; i < n; i++ {
		cands = append(cands, topTestCandidate(t, i, 1_000_000-int64(i), "evm"))
	}
	for i := 0; i < 1000; i++ {
		cands = append(cands, topTestCandidate(t, n+i, 1_000-int64(i), "account"))
	}

	top := NewTopBalances(n)
	for i := range cands {
		top.Add(cands[i])
	}

	fromSnapshot := RichListRows(top.Sorted(), RichListCodes, richListReturned)
	fromFullSort := RichListRows(cands, RichListCodes, richListReturned)

	if len(fromFullSort) != 1000 {
		t.Fatalf("full sort should carry 1000 richlist rows, got %d", len(fromFullSort))
	}
	if len(fromSnapshot) != 0 {
		t.Fatalf("with no margin the snapshot list must be empty here, got %d rows", len(fromSnapshot))
	}

	// Growing N to cover the filler restores the exact list.
	bigger := NewTopBalances(n + 1000)
	for i := range cands {
		bigger.Add(cands[i])
	}
	assertSameRows(t, RichListRows(bigger.Sorted(), RichListCodes, richListReturned), fromFullSort)
}

// TestRichListRowsFiltersAndOrders pins the read-time filter: only the richlist
// codes, only a positive balance, balance desc then address asc, at most k.
func TestRichListRowsFiltersAndOrders(t *testing.T) {
	cands := []TopBalanceCandidate{
		topTestCandidate(t, 1, 500, "account"),
		topTestCandidate(t, 2, 900, "evm"),      // filtered: not a richlist code
		topTestCandidate(t, 3, 0, "account"),    // filtered: zero balance
		topTestCandidate(t, 4, 700, "multisig"), // kept
		topTestCandidate(t, 5, 700, "storageminer"),
		topTestCandidate(t, 6, -5, "account"), // filtered: negative
	}

	got := RichListRows(cands, RichListCodes, -1)
	if len(got) != 3 {
		t.Fatalf("kept %d rows, want 3", len(got))
	}
	if got[0].Addr != cands[3].Addr || got[1].Addr != cands[4].Addr || got[2].Addr != cands[0].Addr {
		t.Fatalf("unexpected order: %v", got)
	}

	if limited := RichListRows(cands, RichListCodes, 2); len(limited) != 2 {
		t.Fatalf("k=2 kept %d rows", len(limited))
	}
}
