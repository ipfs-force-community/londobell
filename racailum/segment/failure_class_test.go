package segment

import (
	"errors"
	"fmt"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"

	"go.uber.org/zap"
)

// 常量：与客服实测的 cali 缺块错误形态一致（blockstore 里找不到某个 HAMT 节点）。
const testMissingStateCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

// caliMissingBlockErr 是线上实测的「缺块」错误原文（§8.2 的 state 类）。
func caliMissingBlockErr() error {
	return errors.New("extracting exec-trace: tipset.Height: 4138627: tipset execution failed: " +
		"making vm: loading state tree: failed to load hamt node: " +
		"blockstore get: ipld: could not find " + testMissingStateCID)
}

// mkTSS 造一组合法高度的 LinkedTipSet（只用到 Height()）。
// BlockHeader.Cid() 会 marshal 地址字段，故必须给 Miner 一个有效的 ID 地址，
// 否则 ToStorageBlock 会 panic（cannot marshal undefined address）。
func mkTSS(t *testing.T, heights ...abi.ChainEpoch) []*common.LinkedTipSet {
	t.Helper()
	mkAddr := func() address.Address {
		a, err := address.NewIDAddress(1)
		if err != nil {
			t.Fatalf("构造 ID 地址失败: %v", err)
		}
		return a
	}
	c, err := cid.Decode(testMissingStateCID)
	if err != nil {
		t.Fatalf("构造占位 CID 失败: %v", err)
	}
	out := make([]*common.LinkedTipSet, 0, len(heights))
	for _, h := range heights {
		blk := &types.BlockHeader{
			Height:                h,
			Miner:                 mkAddr(),
			Ticket:                &types.Ticket{VRFProof: []byte{0}},
			ParentWeight:          types.NewInt(0),
			ParentBaseFee:         types.NewInt(0),
			ParentStateRoot:       c,
			ParentMessageReceipts: c,
			Messages:              c,
		}
		ts, err := types.NewTipSet([]*types.BlockHeader{blk})
		if err != nil {
			t.Fatalf("构造 tipset@%d 失败: %v", h, err)
		}
		out = append(out, &common.LinkedTipSet{TipSet: ts})
	}
	return out
}

// 分类器表驱动：规格表每一行一个正例，外加「未知错误必须归为不可登记」与
// 「transport 类必须不可登记」两类反例。
func TestClassifyPartFailure(t *testing.T) {
	part := mkTSS(t, 10, 11)

	cases := []struct {
		name      string
		msg       string
		wantClass string
		wantOK    bool
		wantCID   string
	}{
		{
			name:      "state/ipld-could-not-find",
			msg:       "extracting exec-trace: tipset.Height: 4138627: tipset execution failed: making vm: loading state tree: failed to load hamt node: blockstore get: ipld: could not find " + testMissingStateCID,
			wantClass: FailureClassState,
			wantOK:    true,
			wantCID:   testMissingStateCID,
		},
		{
			name:      "state/load-hamt-node",
			msg:       "tipset execution failed: loading state tree: failed to load hamt node",
			wantClass: FailureClassState,
			wantOK:    true,
		},
		{
			name:      "state/load-state-tree",
			msg:       "making vm: failed to load state tree: boom",
			wantClass: FailureClassState,
			wantOK:    true,
		},
		{
			name:      "data/not-minimally-encoded",
			msg:       "decode message: not minimally encoded during unmarshal",
			wantClass: FailureClassData,
			wantOK:    true,
		},
		{
			name:      "data/failed-to-unmarshal",
			msg:       "extract tipset: failed to unmarshal cbor payload",
			wantClass: FailureClassData,
			wantOK:    true,
		},
		{
			name:      "transport/async-persist-wait-timeout",
			msg:       "error occurs in async persist: async persist wait timeout after 1h0m0s",
			wantClass: FailureClassTransport,
			wantOK:    false,
		},
		{
			name:      "transport/context-deadline-exceeded",
			msg:       "insert extracted documents from tipsets: context deadline exceeded",
			wantClass: FailureClassTransport,
			wantOK:    false,
		},
		{
			name:      "transport/connection-incomplete-read",
			msg:       "read tcp 10.0.0.1:1234->10.0.0.2:5678: connection reset by peer: incomplete read",
			wantClass: FailureClassTransport,
			wantOK:    false,
		},
		{
			name:      "unknown/must-not-be-recordable",
			msg:       "some totally unexpected failure that matches no rule",
			wantClass: FailureClassUnknown,
			wantOK:    false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pf, ok := classifyPartFailure(part, errors.New(tc.msg))
			if pf.Class != tc.wantClass {
				t.Fatalf("class 期望 %q，实际 %q", tc.wantClass, pf.Class)
			}
			if ok != tc.wantOK {
				t.Fatalf("可登记性期望 %v，实际 %v", tc.wantOK, ok)
			}
			// 与第二返回值同源的不变量：只有 state/data 才可能可登记。
			if ok && !ClassIsRecordable(pf.Class) {
				t.Fatalf("class %q 不该被判为可登记", pf.Class)
			}
			if pf.FromEpoch != 10 || pf.ToEpoch != 11 {
				t.Fatalf("区间期望 [10,11]，实际 [%d,%d]", pf.FromEpoch, pf.ToEpoch)
			}
			if tc.wantCID != "" && pf.CID != tc.wantCID {
				t.Fatalf("CID 期望 %q，实际 %q", tc.wantCID, pf.CID)
			}
			if pf.ErrDigest == "" {
				t.Fatalf("ErrDigest 不应为空")
			}
		})
	}
}

// transport：作业超时必须用 errors.As 判类型（而非字符串），且必须不可登记。
func TestClassifyPartFailure_JobTimeoutTypeIsTransport(t *testing.T) {
	part := mkTSS(t, 20, 21)
	err := fmt.Errorf("extract job failed after 2 attempt(s): %w",
		&extractJobTimeoutErr{what: "tipset 20", timeout: 0})

	pf, ok := classifyPartFailure(part, err)
	if pf.Class != FailureClassTransport {
		t.Fatalf("作业超时应归 transport，实际 %q", pf.Class)
	}
	if ok {
		t.Fatalf("transport 必须不可登记")
	}
}

// config 为空（默认）时：state 类失败**不登记**（该类别未开启），但它的起始高度仍是阻断面——
// 本轮从它之前截断、下一轮重做；不再整批作废。
func TestPartRunner_EmptySkippableClassesBlocksWithoutRecording(t *testing.T) {
	tss := mkTSS(t, 100, 101, 102)

	var recordCalls int
	r := &partRunner{
		partSizeLimit: 1,
		skippable:     nil, // 默认 = 不登记、不跳过
		log:           zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			if part[0].Height() == 101 {
				return caliMissingBlockErr()
			}
			return nil
		},
		record: func(pf PartFailure) error { recordCalls++; return nil },
	}

	failures, err := r.run(tss)
	if err != nil {
		t.Fatalf("part 失败不再致命（它是阻断面），不应返回 error，实际 %v", err)
	}
	if len(failures) != 1 || failures[0].FromEpoch != 101 {
		t.Fatalf("应记录 1 个阻断面 @101，实际 %+v", failures)
	}
	if failures[0].Kind != BlockedByPersistFailure {
		t.Fatalf("未开启的可登记类应标记为 persist-failure，实际 %q", failures[0].Kind)
	}
	if recordCalls != 0 {
		t.Fatalf("未开启的类别不应写台账，实际写 %d 次", recordCalls)
	}
}

// 第 3 段契约：已登记的洞**不再**当 error 返回（返回 (failures, nil)，发布决策交给上层）；
// 同时验证：失败 part 之后的 part 一律以 persist=false 抽取（不落库）。
func TestPartRunner_RecordedFailureReturnsFailuresWithNilErrorAndDisablesPersist(t *testing.T) {
	tss := mkTSS(t, 100, 101, 102, 103, 104)

	var recordCalls int
	var persistFlags []bool
	r := &partRunner{
		partSizeLimit: 1,
		skippable:     []string{FailureClassState},
		log:           zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			persistFlags = append(persistFlags, persist)
			if part[0].Height() == 101 {
				return caliMissingBlockErr()
			}
			return nil
		},
		record: func(pf PartFailure) error { recordCalls++; return nil },
	}

	failures, err := r.run(tss)
	if err != nil {
		t.Fatalf("已登记的洞不应再返回 error（发布决策交给上层），实际 %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("应登记恰好 1 个 failure，实际 %d", len(failures))
	}
	if recordCalls != 1 {
		t.Fatalf("台账写入期望 1 次，实际 %d", recordCalls)
	}
	if failures[0].Class != FailureClassState || failures[0].FromEpoch != 101 || failures[0].ToEpoch != 101 {
		t.Fatalf("登记内容不符：%+v", failures[0])
	}
	if failures[0].Kind != BlockedByHole {
		t.Fatalf("可登记类应标记为 hole，实际 %q", failures[0].Kind)
	}
	// persist 只对「失败 part 及其之前」为 true：h100、h101 为 true，h102~h104 为 false。
	want := []bool{true, true, false, false, false}
	if len(persistFlags) != len(want) {
		t.Fatalf("extractPart 调用次数期望 %d，实际 %d（%v）", len(want), len(persistFlags), persistFlags)
	}
	for i := range want {
		if persistFlags[i] != want[i] {
			t.Fatalf("第 %d 个 part 的 persist 期望 %v，实际 %v（全部：%v）", i, want[i], persistFlags[i], persistFlags)
		}
	}
}

// 一轮内暴露多个洞：登记多个 failure；第 3 段起不再返回 error（发布决策交给上层）。
func TestPartRunner_RecordsMultipleHolesInOneRound(t *testing.T) {
	tss := mkTSS(t, 200, 201, 202, 203, 204, 205)

	var recordCalls int
	holes := map[abi.ChainEpoch]bool{201: true, 204: true}
	r := &partRunner{
		partSizeLimit: 1,
		skippable:     []string{FailureClassState},
		log:           zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			if holes[part[0].Height()] {
				return caliMissingBlockErr()
			}
			return nil
		},
		record: func(pf PartFailure) error { recordCalls++; return nil },
	}

	failures, err := r.run(tss)
	if err != nil {
		t.Fatalf("已登记的洞不应返回 error，实际 %v", err)
	}
	if len(failures) != 2 || recordCalls != 2 {
		t.Fatalf("期望一轮内登记 2 个洞（failures=%d recordCalls=%d）", len(failures), recordCalls)
	}
}

// 护栏：进入「只抽取不落库」后，未发布区间超阈值即停止继续往后抽。
func TestPartRunner_GuardStopsFurtherExtraction(t *testing.T) {
	tss := mkTSS(t, 300, 301, 302, 303)

	var calls int
	r := &partRunner{
		partSizeLimit:  1,
		skippable:      []string{FailureClassState},
		maxUnpublished: 5,
		log:            zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			calls++
			if part[0].Height() == 301 {
				return caliMissingBlockErr()
			}
			return nil
		},
		record:           func(pf PartFailure) error { return nil },
		unpublishedRange: func() (abi.ChainEpoch, bool) { return 10, true }, // 10 > 5
	}

	failures, err := r.run(tss)
	if err != nil {
		t.Fatalf("已登记的洞不应返回 error，实际 %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("应登记 1 个洞，实际 %d", len(failures))
	}
	// 护栏在 h301 登记后立即触发：h302/h303 不应再被抽取。
	if calls != 2 {
		t.Fatalf("护栏触发后不应继续抽取，extractPart 实际被调 %d 次", calls)
	}
}

// 护栏在「取不到水位」时不生效（宁可不拦，也不拿不确定值拦）。
func TestPartRunner_GuardDisabledWhenRangeUnknown(t *testing.T) {
	tss := mkTSS(t, 400, 401, 402)

	var calls int
	r := &partRunner{
		partSizeLimit:  1,
		skippable:      []string{FailureClassState},
		maxUnpublished: 1,
		log:            zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			calls++
			if part[0].Height() == 400 {
				return caliMissingBlockErr()
			}
			return nil
		},
		record:           func(pf PartFailure) error { return nil },
		unpublishedRange: func() (abi.ChainEpoch, bool) { return 0, false }, // 取不到
	}

	if _, err := r.run(tss); err != nil {
		t.Fatalf("已登记的洞不应返回 error，实际 %v", err)
	}
	if calls != 3 {
		t.Fatalf("取不到水位时护栏不应拦，extractPart 期望被调 3 次，实际 %d", calls)
	}
}

// 铁律：台账写失败 ⇒ 该 part 按「不可登记」处理，整批作废（不登记、不跳过）。
func TestPartRunner_LedgerWriteFailureAborts(t *testing.T) {
	tss := mkTSS(t, 500, 501)

	var recordCalls int
	r := &partRunner{
		partSizeLimit: 1,
		skippable:     []string{FailureClassState},
		log:           zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			if part[0].Height() == 501 {
				return caliMissingBlockErr()
			}
			return nil
		},
		record: func(pf PartFailure) error { recordCalls++; return errors.New("mongo 落库失败") },
	}

	failures, err := r.run(tss)
	if err == nil {
		t.Fatal("台账写失败必须导致整批作废")
	}
	if len(failures) != 0 {
		t.Fatalf("台账写失败时不应登记任何 failure，实际 %d", len(failures))
	}
	if recordCalls != 1 {
		t.Fatalf("期望尝试写台账 1 次，实际 %d", recordCalls)
	}
}

// transport 类：既不登记台账（不可登记），也不再整批作废——它是阻断面（persist-failure），
// 本轮从它之前截断、下一轮重做。
func TestPartRunner_TransportIsBlockingBoundaryNotRecorded(t *testing.T) {
	tss := mkTSS(t, 600, 601)

	var recordCalls int
	r := &partRunner{
		partSizeLimit: 1,
		skippable:     []string{FailureClassState, FailureClassData, FailureClassTransport},
		log:           zap.NewNop().Sugar(),
		extractPart: func(part []*common.LinkedTipSet, persist bool) error {
			if part[0].Height() == 601 {
				return fmt.Errorf("extract job failed after 2 attempt(s): %w", &extractJobTimeoutErr{what: "tipset 601"})
			}
			return nil
		},
		record: func(pf PartFailure) error { recordCalls++; return nil },
	}

	failures, err := r.run(tss)
	if err != nil {
		t.Fatalf("transport 类不再整批作废（它是阻断面），不应返回 error，实际 %v", err)
	}
	if len(failures) != 1 || failures[0].FromEpoch != 601 || failures[0].Class != FailureClassTransport {
		t.Fatalf("应记录 1 个 transport 阻断面 @601，实际 %+v", failures)
	}
	if failures[0].Kind != BlockedByPersistFailure {
		t.Fatalf("transport 阻断面应标记为 persist-failure，实际 %q", failures[0].Kind)
	}
	if recordCalls != 0 {
		t.Fatalf("transport 不得登记台账，实际写 %d 次", recordCalls)
	}
}

func TestClassEnabled(t *testing.T) {
	if classEnabled(FailureClassState, nil) {
		t.Fatal("空配置不应启用任何类别")
	}
	if !classEnabled(FailureClassState, []string{FailureClassState}) {
		t.Fatal("state 应被启用")
	}
	if classEnabled(FailureClassData, []string{FailureClassState}) {
		t.Fatal("data 未被启用")
	}
}
