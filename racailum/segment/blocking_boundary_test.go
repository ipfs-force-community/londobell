package segment

import (
	"context"
	"reflect"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

// TestFirstBlockingBoundary 表驱动：纯函数「求第一个阻断面」（洞 ∪ 失败 part，取最早）。
//
// 覆盖：
//   - 无阻断 ⇒ ok=false；
//   - 多个失败 part ⇒ 取最早（入参故意乱序，验证不依赖顺序）；
//   - 洞与失败 part 混合 ⇒ 取最早（洞最早 / 失败 part 最早 两个方向）；
//   - 旧式字面量（Kind 零值）按洞处理，保持既有语义；
//   - 导出包装 FirstBlockingBoundary 与内部实现同源。
func TestFirstBlockingBoundary(t *testing.T) {
	cases := []struct {
		name     string
		fails    []PartFailure
		wantOK   bool
		wantFrom abi.ChainEpoch
		wantKind string
	}{
		{name: "empty-no-blocking", fails: nil, wantOK: false},
		{
			name:     "single-failed-part",
			fails:    []PartFailure{{FromEpoch: 50, ToEpoch: 51, Kind: BlockedByPersistFailure}},
			wantOK:   true,
			wantFrom: 50,
			wantKind: BlockedByPersistFailure,
		},
		{
			// 情形 c：多个失败 part ⇒ 取最早（乱序入参）。
			name: "multiple-failed-parts-take-earliest",
			fails: []PartFailure{
				{FromEpoch: 204, ToEpoch: 205, Kind: BlockedByPersistFailure},
				{FromEpoch: 201, ToEpoch: 203, Kind: BlockedByPersistFailure},
			},
			wantOK:   true,
			wantFrom: 201,
			wantKind: BlockedByPersistFailure,
		},
		{
			// 情形 d：洞与失败 part 混合 ⇒ 取最早（此处洞最早）。
			name: "mixed-hole-earliest",
			fails: []PartFailure{
				{FromEpoch: 303, ToEpoch: 304, Kind: BlockedByPersistFailure},
				{FromEpoch: 301, ToEpoch: 301, Kind: BlockedByHole},
			},
			wantOK:   true,
			wantFrom: 301,
			wantKind: BlockedByHole,
		},
		{
			// 情形 d：洞与失败 part 混合 ⇒ 取最早（此处失败 part 最早）。
			name: "mixed-failure-earliest",
			fails: []PartFailure{
				{FromEpoch: 401, ToEpoch: 401, Kind: BlockedByHole},
				{FromEpoch: 399, ToEpoch: 400, Kind: BlockedByPersistFailure},
			},
			wantOK:   true,
			wantFrom: 399,
			wantKind: BlockedByPersistFailure,
		},
		{
			name:     "empty-kind-defaults-to-hole",
			fails:    []PartFailure{{FromEpoch: 7, ToEpoch: 8}},
			wantOK:   true,
			wantFrom: 7,
			wantKind: BlockedByHole,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := firstBlockingBoundary(tc.fails)
			if ok != tc.wantOK {
				t.Fatalf("ok 期望 %v，实际 %v", tc.wantOK, ok)
			}
			if !tc.wantOK {
				return
			}
			if got.FromEpoch != tc.wantFrom || got.Kind != tc.wantKind {
				t.Fatalf("阻断面期望 (from=%d,kind=%s)，实际 (from=%d,kind=%s)",
					tc.wantFrom, tc.wantKind, got.FromEpoch, got.Kind)
			}
			// 导出包装必须与内部实现一致（口径同 PrefixLenBeforeFirstHole）。
			gotX, okX := FirstBlockingBoundary(tc.fails)
			if okX != tc.wantOK || gotX != got {
				t.Fatalf("FirstBlockingBoundary 与内部实现不一致: (from=%d,kind=%s)/%v vs (from=%d,kind=%s)/%v",
					gotX.FromEpoch, gotX.Kind, okX, got.FromEpoch, got.Kind, ok)
			}
		})
	}
}

// TestCommitPrefix_BlockingBoundaryTruncatesPublish 表驱动覆盖「阻断面截断发布」6 种情形：
//
//	a) 阻断面在批首 ⇒ 不发布、水位不动、Aggregate 不被调用；
//	b) 阻断面在批中 ⇒ 发布 range 精确 = [批首, 阻断面-1]，Aggregate 只收到该前缀；
//	c) 多个失败 part ⇒ 取最早；
//	d) 洞与失败 part 混合 ⇒ 取最早；
//	e) 无阻断 ⇒ 整批发布（与现状一致）；
//	f) 阻断面之后还有已成功的 part ⇒ 仍只发布到阻断面之前，绝不跳过。
//
// 同时锁住红线：Aggregate 收到的任何高度都 < 阻断面起始高度（绝不跨阻断面发布）。
func TestCommitPrefix_BlockingBoundaryTruncatesPublish(t *testing.T) {
	const notPublished abi.ChainEpoch = -1

	cases := []struct {
		name          string
		heights       []abi.ChainEpoch
		fails         []PartFailure
		wantErr       bool
		wantAgg       []abi.ChainEpoch // nil = Aggregate 不得被调用
		wantPublished abi.ChainEpoch   // notPublished = 不得发布（水位不动）
	}{
		{
			name:          "a-blocked-at-batch-head-publishes-nothing",
			heights:       []abi.ChainEpoch{10, 11, 12},
			fails:         []PartFailure{{FromEpoch: 10, ToEpoch: 11, Class: FailureClassState, Kind: BlockedByHole}},
			wantAgg:       nil,
			wantPublished: notPublished,
		},
		{
			name:          "b-blocked-in-middle-publishes-prefix-only",
			heights:       []abi.ChainEpoch{100, 101, 102, 103, 104},
			fails:         []PartFailure{{FromEpoch: 103, ToEpoch: 103, Class: FailureClassTransport, Kind: BlockedByPersistFailure}},
			wantAgg:       []abi.ChainEpoch{100, 101, 102},
			wantPublished: 102,
		},
		{
			name:    "c-multiple-failed-parts-take-earliest",
			heights: []abi.ChainEpoch{200, 201, 202, 203, 204, 205},
			fails: []PartFailure{
				{FromEpoch: 204, ToEpoch: 204, Kind: BlockedByPersistFailure},
				{FromEpoch: 201, ToEpoch: 202, Kind: BlockedByPersistFailure},
			},
			wantAgg:       []abi.ChainEpoch{200},
			wantPublished: 200,
		},
		{
			name:    "d-hole-and-failed-part-mixed-take-earliest",
			heights: []abi.ChainEpoch{300, 301, 302, 303, 304, 305},
			fails: []PartFailure{
				{FromEpoch: 303, ToEpoch: 303, Class: FailureClassState, Kind: BlockedByHole},
				{FromEpoch: 301, ToEpoch: 302, Class: FailureClassTransport, Kind: BlockedByPersistFailure},
			},
			wantAgg:       []abi.ChainEpoch{300},
			wantPublished: 300,
		},
		{
			name:          "e-no-blocking-publishes-whole-batch",
			heights:       []abi.ChainEpoch{1, 2, 3, 4},
			fails:         nil,
			wantAgg:       []abi.ChainEpoch{1, 2, 3, 4},
			wantPublished: 4,
		},
		{
			// f：402 是阻断面；403/404/405 即便已成功也绝不发布过去。
			name:          "f-successful-parts-after-boundary-are-not-skipped-over",
			heights:       []abi.ChainEpoch{400, 401, 402, 403, 404, 405},
			fails:         []PartFailure{{FromEpoch: 402, ToEpoch: 402, Kind: BlockedByPersistFailure}},
			wantAgg:       []abi.ChainEpoch{400, 401},
			wantPublished: 401,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tss := mkTSS(t, tc.heights...)

			var (
				aggHeights []abi.ChainEpoch
				aggCalled  bool
				fhHeight   abi.ChainEpoch = notPublished
				shHeight   abi.ChainEpoch = notPublished
				ubHeight   abi.ChainEpoch = notPublished
			)
			deps := prefixCommitDeps{
				aggregate: func(_ context.Context, p []*common.LinkedTipSet) error {
					aggCalled = true
					for _, ts := range p {
						aggHeights = append(aggHeights, ts.Height())
					}
					return nil
				},
				saveFinalHeight: func(_ context.Context, h *common.LinkedTipSet) error { fhHeight = h.Height(); return nil },
				saveStateHeight: func(_ context.Context, h *common.LinkedTipSet) error { shHeight = h.Height(); return nil },
				updateBoundary:  func(_ context.Context, h *common.LinkedTipSet) error { ubHeight = h.Height(); return nil },
			}

			err := commitPrefix(context.Background(), tss, tc.fails, deps)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err 期望 %v，实际 %v", tc.wantErr, err)
			}

			if tc.wantAgg == nil {
				if aggCalled {
					t.Fatalf("不得调用 Aggregate，实际收到 %v", aggHeights)
				}
			} else if !reflect.DeepEqual(aggHeights, tc.wantAgg) {
				t.Fatalf("Aggregate 收到的高度期望 %v，实际 %v", tc.wantAgg, aggHeights)
			}

			// 红线：Aggregate 收到的任何高度都必须严格小于阻断面起始高度。
			if b, ok := firstBlockingBoundary(tc.fails); ok {
				for _, h := range aggHeights {
					if h >= b.FromEpoch {
						t.Fatalf("Aggregate 收到阻断面 %d 或其之后的高度 %d ⇒ 跨阻断面发布（红线被突破）", b.FromEpoch, h)
					}
				}
			}

			if tc.wantPublished == notPublished {
				if fhHeight != notPublished || ubHeight != notPublished || shHeight != notPublished {
					t.Fatalf("不得发布/推水位，实际 final=%d state=%d boundary=%d", fhHeight, shHeight, ubHeight)
				}
			} else {
				if fhHeight != tc.wantPublished {
					t.Fatalf("SaveFinalHeight 应收到前缀末 %d，实际 %d", tc.wantPublished, fhHeight)
				}
				if ubHeight != tc.wantPublished {
					t.Fatalf("updateBoundary 应收到前缀末 %d，实际 %d", tc.wantPublished, ubHeight)
				}
				if shHeight != notPublished {
					t.Fatalf("非 OnlyExtractState 时不应调用 SaveStateFinalHeight，实际 %d", shHeight)
				}
			}
		})
	}
}
