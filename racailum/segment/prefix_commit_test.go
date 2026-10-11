package segment

import (
	"context"
	"reflect"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

// prefixLenBeforeFirstHole 表驱动：无洞 / 洞在批首 / 洞在中间 / 多个洞（取最早）。
func TestPrefixLenBeforeFirstHole(t *testing.T) {
	cases := []struct {
		name    string
		heights []abi.ChainEpoch
		fails   []PartFailure
		want    int
	}{
		{
			name:    "no-holes-returns-len",
			heights: []abi.ChainEpoch{10, 11, 12, 13, 14},
			fails:   nil,
			want:    5,
		},
		{
			name:    "hole-at-batch-head-returns-zero",
			heights: []abi.ChainEpoch{10, 11, 12, 13},
			fails:   []PartFailure{{FromEpoch: 10, ToEpoch: 11}},
			want:    0,
		},
		{
			name:    "hole-in-middle-returns-heights-before-it",
			heights: []abi.ChainEpoch{10, 11, 12, 13, 14, 15},
			fails:   []PartFailure{{FromEpoch: 13, ToEpoch: 14}},
			want:    3, // 10,11,12
		},
		{
			name:    "multiple-holes-take-earliest",
			heights: []abi.ChainEpoch{20, 21, 22, 23, 24, 25, 26},
			fails:   []PartFailure{{FromEpoch: 24, ToEpoch: 24}, {FromEpoch: 21, ToEpoch: 22}},
			want:    1, // 只有 20 < 21
		},
		{
			name:    "hole-start-equals-second-height-strictly-less",
			heights: []abi.ChainEpoch{30, 31, 32},
			fails:   []PartFailure{{FromEpoch: 31, ToEpoch: 32}},
			want:    1, // 严格小于 31 ⇒ 只有 30
		},
		{
			name:    "hole-beyond-batch-returns-len",
			heights: []abi.ChainEpoch{40, 41, 42},
			fails:   []PartFailure{{FromEpoch: 99, ToEpoch: 100}},
			want:    3,
		},
		{
			name:    "empty-tss-with-holes-returns-zero",
			heights: nil,
			fails:   []PartFailure{{FromEpoch: 5, ToEpoch: 5}},
			want:    0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var tss []*common.LinkedTipSet
			if tc.heights != nil {
				tss = mkTSS(t, tc.heights...)
			}
			if got := prefixLenBeforeFirstHole(tss, tc.fails); got != tc.want {
				t.Fatalf("prefixLen 期望 %d，实际 %d", tc.want, got)
			}
		})
	}
}

// ★ 红线用例（本段最关键）：中间有洞时，Aggregate 收到的切片必须**恰好等于前缀**、
// 不包含任何洞后高度；SaveFinalHeight / updateBoundary 必须收到前缀末——而不是整批末。
// 这就直接证明了「跨洞发布不可能发生」。
func TestCommitPrefix_MiddleHolePublishesOnlyPrefix(t *testing.T) {
	tss := mkTSS(t, 100, 101, 102, 103, 104) // 5 个高度，洞在 103
	fails := []PartFailure{{FromEpoch: 103, ToEpoch: 103, Class: FailureClassState}}

	var aggHeights []abi.ChainEpoch
	var fhHeight abi.ChainEpoch = -1
	var ubHeight abi.ChainEpoch = -1
	var stateHeight abi.ChainEpoch = -1

	deps := prefixCommitDeps{
		aggregate: func(_ context.Context, p []*common.LinkedTipSet) error {
			for _, ts := range p {
				aggHeights = append(aggHeights, ts.Height())
			}
			return nil
		},
		saveFinalHeight: func(_ context.Context, h *common.LinkedTipSet) error {
			fhHeight = h.Height()
			return nil
		},
		saveStateHeight: func(_ context.Context, h *common.LinkedTipSet) error {
			stateHeight = h.Height()
			return nil
		},
		updateBoundary: func(_ context.Context, h *common.LinkedTipSet) error {
			ubHeight = h.Height()
			return nil
		},
	}

	if err := commitPrefix(context.Background(), tss, fails, deps); err != nil {
		t.Fatalf("前缀非空时不应报错，实际 %v", err)
	}

	// Aggregate 收到的切片必须「恰好等于前缀」：高度严格小于 103 的三个。
	wantAgg := []abi.ChainEpoch{100, 101, 102}
	if !reflect.DeepEqual(aggHeights, wantAgg) {
		t.Fatalf("Aggregate 收到的高度期望 %v，实际 %v", wantAgg, aggHeights)
	}
	// 且绝不能包含任何洞后（≥103）的高度。
	for _, h := range aggHeights {
		if h >= 103 {
			t.Fatalf("Aggregate 收到洞后高度 %d ⇒ 跨洞发布派生表（红线被突破）", h)
		}
	}
	// 水位与段边界必须只推到前缀末 102，而不是整批末 104。
	if fhHeight != 102 {
		t.Fatalf("SaveFinalHeight 应收到前缀末 102，实际 %d", fhHeight)
	}
	if fhHeight == tss[len(tss)-1].Height() {
		t.Fatalf("SaveFinalHeight 收到了整批末 %d ⇒ 跨洞前移水位（红线被突破）", fhHeight)
	}
	if ubHeight != 102 {
		t.Fatalf("updateBoundary 应收到前缀末 102，实际 %d", ubHeight)
	}
	if stateHeight != -1 {
		t.Fatalf("非 OnlyExtractState 时不应调用 SaveStateFinalHeight，实际收到 %d", stateHeight)
	}
}

// 兼容性红线：fails 为空时 prefix = 整批，行为与改动前完全一致。
func TestCommitPrefix_NoHolesPublishesWholeBatch(t *testing.T) {
	tss := mkTSS(t, 1, 2, 3, 4)

	var aggLen int
	var fhHeight abi.ChainEpoch = -1
	var ubHeight abi.ChainEpoch = -1

	deps := prefixCommitDeps{
		aggregate: func(_ context.Context, p []*common.LinkedTipSet) error {
			aggLen = len(p)
			return nil
		},
		saveFinalHeight: func(_ context.Context, h *common.LinkedTipSet) error { fhHeight = h.Height(); return nil },
		saveStateHeight: func(_ context.Context, _ *common.LinkedTipSet) error { return nil },
		updateBoundary:  func(_ context.Context, h *common.LinkedTipSet) error { ubHeight = h.Height(); return nil },
	}

	if err := commitPrefix(context.Background(), tss, nil, deps); err != nil {
		t.Fatalf("无洞时不应报错，实际 %v", err)
	}
	if aggLen != len(tss) {
		t.Fatalf("无洞时 Aggregate 应收到整批 %d 个，实际 %d", len(tss), aggLen)
	}
	if fhHeight != 4 || ubHeight != 4 {
		t.Fatalf("无洞时水位/边界应推到整批末 4，实际 final=%d boundary=%d", fhHeight, ubHeight)
	}
}

// 洞在批首：没有任何可安全发布的部分 ⇒ 返回 error、不调用任何发布依赖（不动水位）。
func TestCommitPrefix_HoleAtHeadPublishesNothing(t *testing.T) {
	tss := mkTSS(t, 10, 11, 12)
	fails := []PartFailure{{FromEpoch: 10, ToEpoch: 11, Class: FailureClassState}}

	var calls int
	deps := prefixCommitDeps{
		aggregate:       func(_ context.Context, _ []*common.LinkedTipSet) error { calls++; return nil },
		saveFinalHeight: func(_ context.Context, _ *common.LinkedTipSet) error { calls++; return nil },
		saveStateHeight: func(_ context.Context, _ *common.LinkedTipSet) error { calls++; return nil },
		updateBoundary:  func(_ context.Context, _ *common.LinkedTipSet) error { calls++; return nil },
	}

	if err := commitPrefix(context.Background(), tss, fails, deps); err == nil {
		t.Fatal("洞在批首必须返回 error（保持不动水位）")
	}
	if calls != 0 {
		t.Fatalf("洞在批首不应调用任何发布依赖，实际调用 %d 次", calls)
	}
}

// OnlyExtractState 模式：推前缀末走 SaveStateFinalHeight（而不是 SaveFinalHeight），且仍只吃前缀。
func TestCommitPrefix_StateOnlyPublishesPrefixViaStateHeight(t *testing.T) {
	tss := mkTSS(t, 20, 21, 22, 23)
	fails := []PartFailure{{FromEpoch: 22, ToEpoch: 23, Class: FailureClassState}}

	var fhUsed, shUsed bool
	var shHeight abi.ChainEpoch = -1
	deps := prefixCommitDeps{
		aggregate:       func(_ context.Context, _ []*common.LinkedTipSet) error { return nil },
		saveFinalHeight: func(_ context.Context, _ *common.LinkedTipSet) error { fhUsed = true; return nil },
		saveStateHeight: func(_ context.Context, h *common.LinkedTipSet) error {
			shUsed = true
			shHeight = h.Height()
			return nil
		},
		updateBoundary:   func(_ context.Context, _ *common.LinkedTipSet) error { return nil },
		onlyExtractState: true,
		saveStateEnabled: true,
	}

	if err := commitPrefix(context.Background(), tss, fails, deps); err != nil {
		t.Fatalf("不应报错，实际 %v", err)
	}
	if fhUsed {
		t.Fatal("OnlyExtractState 时不应走 SaveFinalHeight")
	}
	if !shUsed || shHeight != 21 {
		t.Fatalf("应走 SaveStateFinalHeight 且收到前缀末 21，实际 used=%v height=%d", shUsed, shHeight)
	}
}

// TestPrefixLenBeforeFirstHole_ExportedWrapper 锁住导出包装与内部实现同源。
// 存在的理由：ra.go 的离线抽取路径（WalkExtract）跨包复用这条判据，
// 若包装与内部实现漂移，离线路径就会绕开「绝不跨洞」这条红线。
func TestPrefixLenBeforeFirstHole_ExportedWrapper(t *testing.T) {
	tss := mkTSS(t, 100, 101, 102, 103)

	cases := []struct {
		name  string
		fails []PartFailure
		want  int
	}{
		{"无洞取整批", nil, 4},
		{"洞在批首取 0", []PartFailure{{FromEpoch: 100, ToEpoch: 100}}, 0},
		{"洞在中间取洞前高度数", []PartFailure{{FromEpoch: 102, ToEpoch: 102}}, 2},
		{"多个洞取最早", []PartFailure{{FromEpoch: 103, ToEpoch: 103}, {FromEpoch: 101, ToEpoch: 101}}, 1},
	}
	for _, c := range cases {
		if got := PrefixLenBeforeFirstHole(tss, c.fails); got != c.want {
			t.Fatalf("%s: PrefixLenBeforeFirstHole = %d, want %d", c.name, got, c.want)
		}
	}
}
