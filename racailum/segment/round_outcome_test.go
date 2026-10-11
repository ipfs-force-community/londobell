package segment

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
)

// roundOutcomeFromFailures 是「本轮是否被阻断面截断」的纯判据（供 RaCailum.Run 决定轮间退避）。
// 表驱动 + 一条不变量：Truncated ⟺ 可发布前缀短于整批（与 prefixLenBeforeFirstHole 同源）。
func TestRoundOutcomeFromFailures(t *testing.T) {
	tss := mkTSS(t, 10, 11, 12, 13, 14)

	cases := []struct {
		name      string
		fails     []PartFailure
		wantTrunc bool
		wantAt    abi.ChainEpoch
	}{
		{
			name:      "无阻断⇒未截断",
			fails:     nil,
			wantTrunc: false,
			wantAt:    -1,
		},
		{
			name:      "洞在批中⇒截断并带起点",
			fails:     []PartFailure{{FromEpoch: 12, ToEpoch: 13, Kind: BlockedByHole}},
			wantTrunc: true,
			wantAt:    12,
		},
		{
			name:      "失败 part 在批首⇒截断",
			fails:     []PartFailure{{FromEpoch: 10, ToEpoch: 10, Kind: BlockedByPersistFailure}},
			wantTrunc: true,
			wantAt:    10,
		},
		{
			name:      "多个阻断⇒取最早",
			fails:     []PartFailure{{FromEpoch: 13, ToEpoch: 13}, {FromEpoch: 11, ToEpoch: 12}},
			wantTrunc: true,
			wantAt:    11,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := roundOutcomeFromFailures(tc.fails)
			if got.Truncated != tc.wantTrunc {
				t.Fatalf("Truncated 期望 %v，实际 %v", tc.wantTrunc, got.Truncated)
			}
			if tc.wantTrunc && got.BlockedAt != tc.wantAt {
				t.Fatalf("BlockedAt 期望 %d，实际 %d", tc.wantAt, got.BlockedAt)
			}

			// 不变量：截断 ⟺ 前缀短于整批。这条把 RoundOutcome 与发布判据钉在一起，
			// 防止「判了截断但实际全批发布」（或反之）这类漂移。
			shortfall := prefixLenBeforeFirstHole(tss, tc.fails) < len(tss)
			if got.Truncated != shortfall {
				t.Fatalf("Truncated=%v 与「前缀短于整批」=%v 不一致（前缀=%d, 整批=%d）",
					got.Truncated, shortfall, prefixLenBeforeFirstHole(tss, tc.fails), len(tss))
			}
		})
	}
}
