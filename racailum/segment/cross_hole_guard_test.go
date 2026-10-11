package segment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
)

// 运维手动聚合入口（RaCailum.Aggregate）的跨洞防护：CheckNoCrossHoleAggregation 的
// 「四种区间关系」表驱动。台账内容以注入的 query 模拟，不连 mongo；
// 断言落到「放行（nil）/ 拒绝（error 且正文带洞的区间）」这两个可观测结果上。
//
// 边界语义：洞恰好落在 lo 或 hi（相触即算相交）也必须拒绝——用闭区间判据，不是开区间。
func TestCheckNoCrossHoleAggregation_RangeRelations(t *testing.T) {
	cases := []struct {
		name         string
		holes        []HoleSpan
		lo, hi       abi.ChainEpoch
		wantRefuse   bool
		wantContains []string // 拒绝时错误正文必须包含的子串（洞的高度区间）
	}{
		{
			name:       "区间内无洞⇒放行",
			holes:      nil,
			lo:         100,
			hi:         200,
			wantRefuse: false,
		},
		{
			name:         "洞在区间中间⇒拒绝",
			holes:        []HoleSpan{{FromEpoch: 140, ToEpoch: 150, Class: FailureClassState}},
			lo:           100,
			hi:           200,
			wantRefuse:   true,
			wantContains: []string{"refused", "[140,150]"},
		},
		{
			name:         "洞恰好落在 lo 边界⇒拒绝",
			holes:        []HoleSpan{{FromEpoch: 100, ToEpoch: 105, Class: FailureClassState}},
			lo:           100,
			hi:           200,
			wantRefuse:   true,
			wantContains: []string{"[100,105]"},
		},
		{
			name:         "洞恰好落在 hi 边界⇒拒绝",
			holes:        []HoleSpan{{FromEpoch: 195, ToEpoch: 200, Class: FailureClassState}},
			lo:           100,
			hi:           200,
			wantRefuse:   true,
			wantContains: []string{"[195,200]"},
		},
		{
			name:       "区间与洞完全不重叠（洞在区间之前）⇒放行",
			holes:      []HoleSpan{{FromEpoch: 10, ToEpoch: 20, Class: FailureClassState}},
			lo:         100,
			hi:         200,
			wantRefuse: false,
		},
		{
			name:       "区间与洞完全不重叠（洞在区间之后）⇒放行",
			holes:      []HoleSpan{{FromEpoch: 300, ToEpoch: 310, Class: FailureClassState}},
			lo:         100,
			hi:         200,
			wantRefuse: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			q := func(_ context.Context, _, _ abi.ChainEpoch) ([]HoleSpan, error) {
				return tc.holes, nil
			}

			err := CheckNoCrossHoleAggregation(context.Background(), q, tc.lo, tc.hi)

			if tc.wantRefuse {
				if err == nil {
					t.Fatalf("期望拒绝（返回 error），实际放行")
				}
				for _, sub := range tc.wantContains {
					if !strings.Contains(err.Error(), sub) {
						t.Fatalf("拒绝错误正文应含 %q（洞的区间），实际 %q", sub, err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("期望放行（返回 nil），实际 %v", err)
			}
		})
	}
}

// 失败关闭：台账查询失败 ⇒ 拒绝（返回 error 并附原始错误），绝不因「查不到」而放行。
// 语义理由：查不到台账 = 不知道有没有洞，放行等于赌数据正确性。
func TestCheckNoCrossHoleAggregation_FailsClosedOnQueryError(t *testing.T) {
	sentinel := errors.New("mongo unreachable")
	q := func(_ context.Context, _, _ abi.ChainEpoch) ([]HoleSpan, error) {
		return nil, sentinel
	}

	err := CheckNoCrossHoleAggregation(context.Background(), q, 100, 200)
	if err == nil {
		t.Fatal("台账查询失败必须失败关闭（返回 error），不得放行")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("错误必须附原始错误（errors.Is 应命中 sentinel），实际 %v", err)
	}
}

// 默认台账查询实现（RegisteredHolesInRange）在写库不可用时必须返回 error（而非空结果）。
// 这是「失败关闭」在真实（非注入）实现里的落点：查询本身不产生「零洞」的假真理。
func TestRegisteredHolesInRange_NilDBFailsClosed(t *testing.T) {
	s := &Segment{} // db 为 nil（如同写库连不上）
	if _, err := s.RegisteredHolesInRange(context.Background(), 0, 100); err == nil {
		t.Fatal("写库为 nil 时必须返回 error（失败关闭），不得返回空结果放行")
	}
}
