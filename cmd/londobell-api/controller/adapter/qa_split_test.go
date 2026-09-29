package adapter

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/filecoin-project/go-state-types/abi"
	sbig "github.com/filecoin-project/go-state-types/big"
	miner19 "github.com/filecoin-project/go-state-types/builtin/v19/miner"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
)

// 32GiB 扇区。
const testSectorSize = uint64(34359738368)

const (
	tSimpleQAPower = uint64(1) // miner19.SIMPLE_QA_POWER
	tFullQAPower   = uint64(2) // miner19.FULL_QA_POWER
	tBothFlags     = uint64(3) // NV29 之后新扇区实测 flags = 0x3
)

func bint(v int64) *big.Int { return big.NewInt(v) }

// smul(f) = testSectorSize * f
func smul(f int64) *big.Int {
	return new(big.Int).Mul(new(big.Int).SetUint64(testSectorSize), big.NewInt(f))
}

func smulDiv(f, d int64) *big.Int { return new(big.Int).Div(smul(f), big.NewInt(d)) }

type qaWant struct {
	full bool
	qa   *big.Int // nil = 不断言
	vdc  *big.Int
	dc   *big.Int
	cc   *big.Int
}

func TestQASplit(t *testing.T) {
	// 与 filscan_backend/pkg/londobell/qa_split_test.go 是同一张表（同源实现必须逐行等价）。
	tests := []struct {
		name  string
		flags uint64
		pbe   int64
		act   int64
		exp   int64
		dw    *big.Int
		vdw   *big.Int
		want  qaWant
	}{
		{
			name: "full_flag_vdw_zero_method37_upgraded_cc",
			flags: tFullQAPower, pbe: 1000, act: 1000, exp: 2000000,
			dw: bint(0), vdw: bint(0),
			want: qaWant{true, smul(10), bint(0), bint(0), smul(10)},
		},
		{
			name: "full_flag_vdw_full_space_new_sector",
			flags: tBothFlags, pbe: 1000, act: 1000, exp: 2000000,
			dw: bint(0), vdw: smul(1999000),
			want: qaWant{true, smul(10), bint(0), bint(0), smul(10)},
		},
		{
			name: "vdw_covers_full_space_without_flag",
			flags: tSimpleQAPower, pbe: 1000, act: 1000, exp: 2000000,
			dw: bint(0), vdw: smul(1999000),
			want: qaWant{true, smul(10), bint(0), bint(0), smul(10)},
		},
		{
			name: "legacy_cc_no_flag_vdw_zero",
			flags: tSimpleQAPower, pbe: 1000, act: 1000, exp: 2000000,
			dw: bint(0), vdw: bint(0),
			want: qaWant{false, smul(1), bint(0), bint(0), smul(1)},
		},
		{
			name: "half_verified_split",
			flags: tSimpleQAPower, pbe: 1000, act: 1000, exp: 2000000,
			dw: bint(0), vdw: smul(999500),
			want: qaWant{false, smulDiv(11, 2), smul(5), bint(0), smulDiv(1, 2)},
		},
		{
			name: "renewed_pbe_gt_act_uses_exp_minus_pbe",
			flags: uint64(0), pbe: 1000, act: 100, exp: 1100,
			dw: bint(0), vdw: smul(50),
			// qa = 5.5x；桶权重 vdcW=500S、ccW=size*(exp-act)-vdw=950S、all=1450S，
			// 因此 vdc = 65165021042、dc = 0、cc = qa - vdc = 123813539982
			// （与 filscan 侧 modules/pro/syncer/sector_task_test.go 的黄金值一致）。
			want: qaWant{false, smulDiv(11, 2), big.NewInt(65165021042), bint(0), big.NewInt(123813539982)},
		},
		{
			name: "negative_cc_weight_clamped",
			flags: tSimpleQAPower, pbe: 1000, act: 1000, exp: 2000,
			dw: smul(200), vdw: smul(900),
			want: qaWant{false, nil, nil, nil, nil},
		},
		{
			name: "all_weights_zero_falls_back_to_cc",
			flags: tSimpleQAPower, pbe: 1000, act: 2000, exp: 2000,
			dw: bint(0), vdw: bint(0),
			want: qaWant{false, smul(1), bint(0), bint(0), smul(1)},
		},
		{
			name: "exp_le_pbe_falls_back_to_activation",
			flags: tSimpleQAPower, pbe: 3000, act: 1000, exp: 2000,
			dw: bint(0), vdw: bint(0),
			want: qaWant{false, smul(1), bint(0), bint(0), smul(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qa, vdc, dc, cc, full := QASplit(testSectorSize, tt.flags, tt.pbe, tt.act, tt.exp, tt.dw, tt.vdw)

			if qa == nil || vdc == nil || dc == nil || cc == nil {
				t.Fatalf("QASplit 返回了 nil：qa=%v vdc=%v dc=%v cc=%v", qa, vdc, dc, cc)
			}
			if full != tt.want.full {
				t.Errorf("full = %v, want %v", full, tt.want.full)
			}
			if tt.want.qa != nil && qa.Cmp(tt.want.qa) != 0 {
				t.Errorf("qa = %s, want %s", qa, tt.want.qa)
			}
			if tt.want.vdc != nil && vdc.Cmp(tt.want.vdc) != 0 {
				t.Errorf("vdc = %s, want %s", vdc, tt.want.vdc)
			}
			if tt.want.dc != nil && dc.Cmp(tt.want.dc) != 0 {
				t.Errorf("dc = %s, want %s", dc, tt.want.dc)
			}
			if tt.want.cc != nil && cc.Cmp(tt.want.cc) != 0 {
				t.Errorf("cc = %s, want %s", cc, tt.want.cc)
			}

			for name, v := range map[string]*big.Int{"vdc": vdc, "dc": dc, "cc": cc} {
				if v.Sign() < 0 {
					t.Errorf("%s = %s 为负", name, v)
				}
			}
			sum := new(big.Int).Add(vdc, dc)
			sum.Add(sum, cc)
			if sum.Cmp(qa) != 0 {
				t.Errorf("vdc+dc+cc = %s, want qa = %s", sum, qa)
			}
			if full && (vdc.Sign() != 0 || dc.Sign() != 0 || cc.Cmp(qa) != 0) {
				t.Errorf("full 扇区应 (0,0,qa)，实际 vdc=%s dc=%s cc=%s qa=%s", vdc, dc, cc, qa)
			}
		})
	}
}

// TestQASplitMatchesActorQAPowerForSector：qa 必须与 v19 actor 的权威实现逐位一致。
func TestQASplitMatchesActorQAPowerForSector(t *testing.T) {
	cases := []struct {
		name  string
		flags uint64
		pbe   int64
		act   int64
		exp   int64
		dw    *big.Int
		vdw   *big.Int
	}{
		{"cc", tSimpleQAPower, 1000, 1000, 2000000, bint(0), bint(0)},
		{"partial_verified", tSimpleQAPower, 1000, 1000, 2000000, bint(0), smul(999500)},
		{"legacy_deal_weight_only", tSimpleQAPower, 1000, 1000, 2000000, smul(400000), bint(0)},
		{"renewed", 0, 1000, 100, 1100, bint(0), smul(50)},
		{"full_flag", tFullQAPower, 1000, 1000, 2000000, bint(0), bint(0)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			qa, _, _, _, _ := QASplit(testSectorSize, c.flags, c.pbe, c.act, c.exp, c.dw, c.vdw)
			sector := &miner19.SectorOnChainInfo{
				Activation:         abi.ChainEpoch(c.act),
				Expiration:         abi.ChainEpoch(c.exp),
				DealWeight:         sbig.NewFromGo(c.dw),
				VerifiedDealWeight: sbig.NewFromGo(c.vdw),
				PowerBaseEpoch:     abi.ChainEpoch(c.pbe),
				Flags:              miner19.SectorOnChainInfoFlags(c.flags),
			}
			want := miner19.QAPowerForSector(abi.SectorSize(testSectorSize), sector).Int
			if qa.Cmp(want) != 0 {
				t.Errorf("qa = %s, actor QAPowerForSector = %s", qa, want)
			}
		})
	}
}

// TestIsFullQaPowerAgreesWithLotus：本地判定必须与 lotus miner.SectorIsFullQaPower 一致
// （QASplit 用本地规则切桶、model 里回给 filscan 的 FullQaPower 用本地规则，两者不能漂移）。
func TestIsFullQaPowerAgreesWithLotus(t *testing.T) {
	cases := []struct {
		name  string
		flags uint64
		pbe   int64
		act   int64
		exp   int64
		vdw   *big.Int
	}{
		{"full_flag", tFullQAPower, 1000, 1000, 2000000, bint(0)},
		{"both_flags", tBothFlags, 1000, 1000, 2000000, bint(0)},
		{"simple_cc", tSimpleQAPower, 1000, 1000, 2000000, bint(0)},
		{"vdw_full_no_flag", tSimpleQAPower, 1000, 1000, 2000000, smul(1999000)},
		{"vdw_half_no_flag", tSimpleQAPower, 1000, 1000, 2000000, smul(999500)},
		{"vdw_over_full_no_flag", tSimpleQAPower, 1000, 1000, 2000000, smul(2000000)},
		{"renewed_partial", uint64(0), 1000, 100, 1100, smul(50)},
		{"exp_eq_pbe", tSimpleQAPower, 2000000, 1000, 2000000, smul(10)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			local := IsFullQaPower(testSectorSize, c.flags, c.pbe, c.exp, c.vdw)

			sector := &lminer.SectorOnChainInfo{
				SealProof:          abi.RegisteredSealProof_StackedDrg32GiBV1_1,
				Activation:         abi.ChainEpoch(c.act),
				Expiration:         abi.ChainEpoch(c.exp),
				DealWeight:         sbig.NewFromGo(bint(0)),
				VerifiedDealWeight: sbig.NewFromGo(c.vdw),
				PowerBaseEpoch:     abi.ChainEpoch(c.pbe),
				Flags:              miner19.SectorOnChainInfoFlags(c.flags),
			}
			lotusVerdict := SectorIsFullQaPower(sector)
			if local != lotusVerdict {
				t.Errorf("本地 IsFullQaPower=%v, lotus SectorIsFullQaPower=%v", local, lotusVerdict)
			}
		})
	}
}

// TestComputeQAPower 端到端：一个 NV29 新扇区（0x3，10x）+ 一个老 50% verified 扇区 +
// 一个老 CC 扇区，三桶之和 = 三扇区 QA 之和，且都是整数值（不再是 decimal 除法截断的结果）。
func TestComputeQAPower(t *testing.T) {
	const (
		act = int64(1000)
		pbe = int64(1000)
		exp = int64(2000000)
	)

	newSector := &lminer.SectorOnChainInfo{
		SealProof:          abi.RegisteredSealProof_StackedDrg32GiBV1_1,
		Activation:         abi.ChainEpoch(act),
		Expiration:         abi.ChainEpoch(exp),
		DealWeight:         sbig.NewFromGo(bint(0)),
		VerifiedDealWeight: sbig.NewFromGo(smul(1999000)),
		PowerBaseEpoch:     abi.ChainEpoch(pbe),
		Flags:              miner19.SectorOnChainInfoFlags(tBothFlags),
	}
	halfVerified := &lminer.SectorOnChainInfo{
		SealProof:          abi.RegisteredSealProof_StackedDrg32GiBV1_1,
		Activation:         abi.ChainEpoch(act),
		Expiration:         abi.ChainEpoch(exp),
		DealWeight:         sbig.NewFromGo(bint(0)),
		VerifiedDealWeight: sbig.NewFromGo(smul(999500)),
		PowerBaseEpoch:     abi.ChainEpoch(pbe),
		Flags:              miner19.SectorOnChainInfoFlags(tSimpleQAPower),
	}
	oldCC := &lminer.SectorOnChainInfo{
		SealProof:          abi.RegisteredSealProof_StackedDrg32GiBV1_1,
		Activation:         abi.ChainEpoch(act),
		Expiration:         abi.ChainEpoch(exp),
		DealWeight:         sbig.NewFromGo(bint(0)),
		VerifiedDealWeight: sbig.NewFromGo(bint(0)),
		PowerBaseEpoch:     abi.ChainEpoch(pbe),
		Flags:              miner19.SectorOnChainInfoFlags(tSimpleQAPower),
	}

	res, sectors := ComputeQAPower(
		[]*lminer.SectorOnChainInfo{newSector, halfVerified, oldCC},
		abi.SectorSize(testSectorSize),
	)

	if len(sectors) != 3 {
		t.Fatalf("sectors = %d, want 3", len(sectors))
	}
	// full 扇区：10x，全归 CC。
	if !sectors[0].FullQaPower || sectors[0].Flags != tBothFlags || sectors[0].PowerBaseEpoch != abi.ChainEpoch(pbe) {
		t.Errorf("新扇区应带 FULL 标志并被判为 full：%+v", sectors[0])
	}
	if sectors[1].FullQaPower || sectors[2].FullQaPower {
		t.Error("老扇区不该被判为 full")
	}

	// 扇区 1：10x（cc=10S）；扇区 2：5.5x（vdc=5S, cc=0.5S）；扇区 3：1x（cc=1S）。
	wantVDC := decimal.NewFromBigInt(smul(5), 0)
	wantDC := decimal.Zero
	wantCC := decimal.NewFromBigInt(smul(10), 0)
	wantCC = wantCC.Add(decimal.NewFromBigInt(smulDiv(1, 2), 0))
	wantCC = wantCC.Add(decimal.NewFromBigInt(smul(1), 0))

	if !res.VDCPower.Equal(wantVDC) {
		t.Errorf("totalVDC = %s, want %s", res.VDCPower, wantVDC)
	}
	if !res.DCPower.Equal(wantDC) {
		t.Errorf("totalDC = %s, want %s", res.DCPower, wantDC)
	}
	if !res.CCPower.Equal(wantCC) {
		t.Errorf("totalCC = %s, want %s", res.CCPower, wantCC)
	}

	// 三桶之和 = 三个扇区 QA 之和（10 + 5.5 + 1 = 16.5 个 size）。
	wantSum := decimal.NewFromBigInt(smulDiv(33, 2), 0)
	gotSum := res.VDCPower.Add(res.DCPower).Add(res.CCPower)
	if !gotSum.Equal(wantSum) {
		t.Errorf("三桶之和 = %s, want %s", gotSum, wantSum)
	}
	for name, v := range map[string]decimal.Decimal{"vdc": res.VDCPower, "dc": res.DCPower, "cc": res.CCPower} {
		if !v.IsInteger() {
			t.Errorf("%s = %s 不是整数（老实现会有 decimal 除法截断的小数）", name, v)
		}
	}
}

// TestComputeQAPowerNoTruncation：权重取值超过 int32/int64 中间量的场景下，结果必须精确
// （老实现一路 .Int64() 往返，属于截断风险）。
func TestComputeQAPowerNoTruncation(t *testing.T) {
	const bigSize = uint64(68719476736) // 64GiB
	const (
		act = int64(1000)
		pbe = int64(1000)
		exp = int64(5000000) // 5 年量级
	)
	vdw := new(big.Int).Mul(new(big.Int).SetUint64(bigSize), big.NewInt(exp-pbe))
	// 取一半空间
	vdw.Div(vdw, bint(2))

	s := &lminer.SectorOnChainInfo{
		SealProof:          abi.RegisteredSealProof_StackedDrg64GiBV1_1,
		Activation:         abi.ChainEpoch(act),
		Expiration:         abi.ChainEpoch(exp),
		DealWeight:         sbig.NewFromGo(bint(0)),
		VerifiedDealWeight: sbig.NewFromGo(vdw),
		PowerBaseEpoch:     abi.ChainEpoch(pbe),
		Flags:              miner19.SectorOnChainInfoFlags(tSimpleQAPower),
	}

	res, _ := ComputeQAPower([]*lminer.SectorOnChainInfo{s}, abi.SectorSize(bigSize))

	// 期望值按占比公式独立重算：
	//   vdcW = 10*vdw，dcW = 0，ccW = size*(exp-act) - vdw，all = vdcW+dcW+ccW
	//   qa = QAPowerForWeight(size, exp-pbe, vdw) = 5.5*bigSize（vdw 占 exp-pbe 空间的一半）
	//   vdc = qa*vdcW/all
	ccW := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).SetUint64(bigSize), big.NewInt(exp-act)), vdw)
	all := new(big.Int).Add(new(big.Int).Mul(vdw, bint(10)), ccW)
	qa := new(big.Int).Mul(new(big.Int).SetUint64(bigSize), bint(11))
	qa.Div(qa, bint(2))
	wantVDC := new(big.Int).Div(new(big.Int).Mul(qa, new(big.Int).Mul(vdw, bint(10))), all)

	if res.VDCPower.Cmp(decimal.NewFromBigInt(wantVDC, 0)) != 0 {
		t.Errorf("totalVDC = %s, want %s", res.VDCPower, wantVDC)
	}
	sum := res.VDCPower.Add(res.DCPower).Add(res.CCPower)
	if sum.Cmp(decimal.NewFromBigInt(qa, 0)) != 0 {
		t.Errorf("三桶之和 = %s, want qa（5.5x）= %s", sum, qa)
	}
	if !res.VDCPower.IsInteger() || !res.CCPower.IsInteger() {
		t.Error("三桶必须是整数")
	}
	// 中间量远超 int64 时不应留下垃圾值（老实现一路 .Int64() 往返）。
	if res.VDCPower.Sign() <= 0 || res.CCPower.Sign() <= 0 {
		t.Errorf("vdc/cc 应为正：vdc=%s cc=%s", res.VDCPower, res.CCPower)
	}
}
