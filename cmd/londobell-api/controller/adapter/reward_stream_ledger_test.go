package adapter

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/big"
	reward19 "github.com/filecoin-project/go-state-types/builtin/v19/reward"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/ipfs/go-cid"
)

// 以下常量是 2026-10-08 对 Calibnet 生产节点（Filecoin.StateReadState "t02" +
// Filecoin.ChainReadObj StreamsRoot）的真实只读抓取，未经任何编造或数字改写：
//
//	height 4137446
//	StreamsRoot = bafy2bzacecowiphulk2rbmjaipwy4xnldycccgd5hlejgp3swt27bz3fs3drw
//	两个活流：1 = 隐式共识流（评估 50%）、2 = 显式服务流（评估 45%）
//	一条 tombstone：3（欠 t0200206 4,062.44 FIL）
//
// stream2 份额表 t0199897 73.7463% / t0200442 26.2537%（两者之和恰为 Denom）。
//
// 注意：go-address 的 ID 地址 payload 是 uvarint（见 go-address v1.2.0
// address.go:534 IDFromAddress 用 varint.FromUvarint），不是大端整数；CT 的前置分析
// 用大端解出 t014260492/t09346060 是错的，这里以 go-address 的权威解码为准。
const (
	caliLedgerEpoch = int64(4137446)

	caliStreamsRootHex = "83828301851b0d2f13f7789f00003b00000fca32dc55c71a003eb34e1b06f05b59d3b20000" +
		"1b0d2f13f7789f0000f68302851b063eb89da4ed0000001a003f188e1b063eb89da4ed00001b063eb89da4ed0000" +
		"844400b49b0c82824400d9990c1b0a3bfece7e770e59824400fa9d0c1b03a4b7e528ecf1a780" +
		"81824400d9990c4b000232988d4f3b80979fbf818203818244008e9c0c4a00dc39af64f134280ebd80"
	caliStreamsRootCID = "bafy2bzacecowiphulk2rbmjaipwy4xnldycccgd5hlejgp3swt27bz3fs3drw"

	caliTotalMinted   = "111148834482778413826301908"
	caliTotalBurn     = "40418620321034185201656"
	caliTotalExplicit = "149071960539977904350978"
	caliAccruedID2    = "16221102090338365805507"

	// 抓取时 stream2 本期已提（ClaimedPeriod）与 stream3 的 tombstone 余额。
	caliClaimedStream2    = "10378062698807228342207"
	caliTombstone3Payable = "4062440348184312549053"

	// 抓取时按 weight 在 4137446 评估的结果：stream1 已触底到 Floor 50%，stream2 恒 45%。
	caliEvalStream1 = "500000000000000000"
	caliEvalStream2 = "450000000000000000"
	// Liability = stream1(0) + stream2(Accrued - Claimed) + tombstone3。
	caliLiability = "9905479739715450012353"
)

func memStore(t *testing.T) adt.Store {
	t.Helper()
	return adt.WrapStore(context.Background(), cbor.NewCborStore(blockstore.NewMemory()))
}

// caliV19State 用真实抓取值构造 v19 f02 actor state：StreamsRoot 直接放回真实
// StreamsRoot 的原始 CBOR 字节（因此 LoadStreams 解出的就是链上那份对象）。
func caliV19State(t *testing.T) reward.State {
	t.Helper()

	store := memStore(t)

	raw, err := hex.DecodeString(caliStreamsRootHex)
	if err != nil {
		t.Fatalf("decode streams root hex: %v", err)
	}
	// 把真实 StreamsRoot 的原始 CBOR 解成 v19 StreamsState 再放回 store：cbor-gen
	// 的编码是规范形式，重新 Put 得到的 CID 必须与链上那个逐位相同，否则 fixture 漂移。
	var streams reward19.StreamsState
	if err := streams.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		t.Fatalf("decode real streams root CBOR: %v", err)
	}
	root, err := store.Put(store.Context(), &streams)
	if err != nil {
		t.Fatalf("put real streams root: %v", err)
	}
	if got := root.String(); got != caliStreamsRootCID {
		t.Fatalf("real streams-root CBOR re-hashed to %s, want %s (fixture drifted from the real object)", got, caliStreamsRootCID)
	}

	st, err := reward.MakeState(store, actorstypes.Version19, big.Zero())
	if err != nil {
		t.Fatalf("make v19 reward state: %v", err)
	}
	state, ok := st.GetState().(*reward19.State)
	if !ok {
		t.Fatalf("v19 state type = %T, want *reward19.State", st.GetState())
	}
	state.StreamsRoot = root
	state.TotalMintedReward = big.MustFromString(caliTotalMinted)
	state.TotalBurnMinted = big.MustFromString(caliTotalBurn)
	state.TotalExplicitMinted = big.MustFromString(caliTotalExplicit)
	state.SWATimelockEpochs = 720
	state.SWAActor, err = address.NewIDAddress(118) // t0200118
	if err != nil {
		t.Fatalf("swa address: %v", err)
	}
	state.Accrued = []reward19.StreamAccrual{{ID: 2, Amount: big.MustFromString(caliAccruedID2)}}

	return st
}

// TestBuildRewardStreamLedgerV19RealSample：v19（NV29 已激活）分支，用真实 Calibnet
// 账本逐字段核对响应（权重定点、金额 attoFIL、退休流 tombstone、待提总额）。
func TestBuildRewardStreamLedgerV19RealSample(t *testing.T) {
	// fixture 是 Calibnet 抓取，受益人地址按测试网前缀 "t" 呈现。
	prev := address.CurrentNetwork
	address.CurrentNetwork = address.Testnet
	t.Cleanup(func() { address.CurrentNetwork = prev })

	res, err := buildRewardStreamLedger(abi.ChainEpoch(caliLedgerEpoch), caliV19State(t))
	if err != nil {
		t.Fatalf("build v19 ledger: %v", err)
	}

	if !res.NV29 {
		t.Fatal("nv29 must be true on a v19 reward actor")
	}
	if res.Epoch != caliLedgerEpoch {
		t.Errorf("epoch = %d, want %d", res.Epoch, caliLedgerEpoch)
	}
	if res.Denom != "1000000000000000000" {
		t.Errorf("denom = %q, want 1e18 fixed-point", res.Denom)
	}
	if len(res.Streams) != 2 {
		t.Fatalf("streams = %d, want 2", len(res.Streams))
	}

	// 隐式流：爆块矿工，无 distribution，故余额字段恒 "0"。
	implicit := res.Streams[0]
	if implicit.ID != 1 || !implicit.Implicit {
		t.Errorf("stream[0] = id %d implicit %v, want id 1 implicit", implicit.ID, implicit.Implicit)
	}
	if implicit.EvaluatedWeight != caliEvalStream1 {
		t.Errorf("stream1 evaluated_weight = %q, want %q", implicit.EvaluatedWeight, caliEvalStream1)
	}
	if implicit.Accrued != "0" || implicit.Payable != "0" || implicit.ClaimedPeriod != "0" {
		t.Errorf("implicit stream must carry no balances, got %+v", implicit)
	}
	if len(implicit.Recipients) != 0 {
		t.Errorf("implicit stream must have no recipients, got %+v", implicit.Recipients)
	}

	// 显式服务流：45% 权重，本期应计来自 State.Accrued。
	explicit := res.Streams[1]
	if explicit.ID != 2 || explicit.Implicit {
		t.Errorf("stream[1] = id %d implicit %v, want id 2 explicit", explicit.ID, explicit.Implicit)
	}
	if explicit.EvaluatedWeight != caliEvalStream2 {
		t.Errorf("stream2 evaluated_weight = %q, want %q", explicit.EvaluatedWeight, caliEvalStream2)
	}
	if explicit.Accrued != caliAccruedID2 {
		t.Errorf("stream2 accrued = %q, want %q", explicit.Accrued, caliAccruedID2)
	}
	if explicit.ClaimedPeriod != caliClaimedStream2 {
		t.Errorf("stream2 claimed_period = %q, want %q", explicit.ClaimedPeriod, caliClaimedStream2)
	}
	if explicit.Payable != "0" {
		t.Errorf("stream2 payable = %q, want 0 (no carried balance at capture)", explicit.Payable)
	}
	if len(explicit.Recipients) != 2 {
		t.Fatalf("stream2 recipients = %d, want 2", len(explicit.Recipients))
	}
	if got := explicit.Recipients[0]; got.Address != "t0199897" ||
		got.Share != "737463126843657817" || got.Payable != "0" || got.ClaimedPeriod != caliClaimedStream2 {
		t.Errorf("stream2 recipient[0] = %+v", got)
	}
	if got := explicit.Recipients[1]; got.Address != "t0200442" ||
		got.Share != "262536873156342183" || got.Payable != "0" || got.ClaimedPeriod != "0" {
		t.Errorf("stream2 recipient[1] = %+v", got)
	}

	if len(res.Tombstones) != 1 {
		t.Fatalf("tombstones = %d, want 1", len(res.Tombstones))
	}
	if tb := res.Tombstones[0]; tb.ID != 3 || len(tb.Recipients) != 1 ||
		tb.Recipients[0].Address != "t0200206" || tb.Recipients[0].Payable != caliTombstone3Payable {
		t.Errorf("tombstone = %+v", tb)
	}

	if res.Liability != caliLiability {
		t.Errorf("liability = %q, want %q", res.Liability, caliLiability)
	}
}

// TestStreamLedgerUnsupportedOnV18：钉住 v18 actor 的真实行为——lotus 的 StreamLedger
// 直接报 "reward streams are unsupported in actors v18"；本端点必须把它翻成
// nv29=false + 空数组，而不是把这个错误抛给调用方（主网当前即此分支）。
func TestStreamLedgerUnsupportedOnV18(t *testing.T) {
	store := memStore(t)
	st, err := reward.MakeState(store, actorstypes.Version18, big.Zero())
	if err != nil {
		t.Fatalf("make v18 reward state: %v", err)
	}

	if _, err := st.StreamLedger(0); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("v18 StreamLedger error = %v, want an \"unsupported\" error", err)
	}

	res, err := buildRewardStreamLedger(abi.ChainEpoch(caliLedgerEpoch), st)
	if err != nil {
		t.Fatalf("v18 build must not error: %v", err)
	}
	if res.NV29 {
		t.Error("v18 must set nv29=false")
	}
	if res.Streams == nil || len(res.Streams) != 0 {
		t.Errorf("v18 streams = %#v, want non-nil empty", res.Streams)
	}
	if res.Tombstones == nil || len(res.Tombstones) != 0 {
		t.Errorf("v18 tombstones = %#v, want non-nil empty", res.Tombstones)
	}
	if res.Liability != "0" {
		t.Errorf("v18 liability = %q, want \"0\"", res.Liability)
	}
	if res.Denom != "1000000000000000000" {
		t.Errorf("v18 denom = %q, want 1e18 fixed-point", res.Denom)
	}
}

// TestBuildRewardStreamLedgerV19FailsLoudly：v19 下读不到 StreamsRoot 必须返回明确
// 错误，不得静默降级成空数据（契约 §8.1「不得静默空数据」）。
func TestBuildRewardStreamLedgerV19FailsLoudly(t *testing.T) {
	store := memStore(t)
	st, err := reward.MakeState(store, actorstypes.Version19, big.Zero())
	if err != nil {
		t.Fatalf("make v19 reward state: %v", err)
	}
	state := st.GetState().(*reward19.State)
	state.StreamsRoot = cid.Undef

	res, err := buildRewardStreamLedger(abi.ChainEpoch(caliLedgerEpoch), st)
	if err == nil {
		t.Fatalf("missing streams root must error, got %+v", res)
	}
}

// TestRewardStreamLedgerJSONShape：钉住契约 §8.1 的线格式（字段名与空数组语义），
// 避免调用方拿到 nil（JSON null）或改名字段。
func TestRewardStreamLedgerJSONShape(t *testing.T) {
	store := memStore(t)
	v18, err := reward.MakeState(store, actorstypes.Version18, big.Zero())
	if err != nil {
		t.Fatalf("make v18 reward state: %v", err)
	}
	v18Res, err := buildRewardStreamLedger(1, v18)
	if err != nil {
		t.Fatalf("build v18: %v", err)
	}
	v18JSON, err := json.Marshal(v18Res)
	if err != nil {
		t.Fatalf("marshal v18: %v", err)
	}
	for _, want := range []string{`"nv29":false`, `"denom":"1000000000000000000"`, `"streams":[]`, `"tombstones":[]`, `"liability":"0"`} {
		if !strings.Contains(string(v18JSON), want) {
			t.Errorf("v18 JSON missing %s: %s", want, v18JSON)
		}
	}

	prev := address.CurrentNetwork
	address.CurrentNetwork = address.Testnet
	t.Cleanup(func() { address.CurrentNetwork = prev })

	v19Res, err := buildRewardStreamLedger(abi.ChainEpoch(caliLedgerEpoch), caliV19State(t))
	if err != nil {
		t.Fatalf("build v19: %v", err)
	}
	v19JSON, err := json.Marshal(v19Res)
	if err != nil {
		t.Fatalf("marshal v19: %v", err)
	}
	for _, want := range []string{
		`"nv29":true`,
		`"evaluated_weight":"` + caliEvalStream2 + `"`,
		`"address":"t0199897"`,
		`"share":"737463126843657817"`,
		`"claimed_period":"` + caliClaimedStream2 + `"`,
		`"liability":"` + caliLiability + `"`,
	} {
		if !strings.Contains(string(v19JSON), want) {
			t.Errorf("v19 JSON missing %s: %s", want, v19JSON)
		}
	}
	if strings.Contains(string(v19JSON), `"streams":null`) {
		t.Errorf("v19 streams must not be null: %s", v19JSON)
	}
}
