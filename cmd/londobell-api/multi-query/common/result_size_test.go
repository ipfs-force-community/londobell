package common

import (
	"errors"
	"fmt"
	"testing"
)

func TestResolveResultSizePolicyPrecedence(t *testing.T) {
	// 默认：配置文件零值 ⇒ 内置默认
	if got := ResolveResultSizePolicy(Config{}); got != DefaultMaxResultBytes {
		t.Fatalf("zero config: got %d, want default %d", got, DefaultMaxResultBytes)
	}

	// 配置文件正数 ⇒ 用该值
	if got := ResolveResultSizePolicy(Config{MaxResultBytes: 1 << 20}); got != 1<<20 {
		t.Fatalf("cfg>0: got %d, want %d", got, 1<<20)
	}

	// 配置文件负数 ⇒ 关闭（0）
	if got := ResolveResultSizePolicy(Config{MaxResultBytes: -1}); got != 0 {
		t.Fatalf("cfg<0: got %d, want 0 (disabled)", got)
	}

	// 环境变量优先于配置文件
	t.Setenv(MaxResultBytesEnv, "4096")
	if got := ResolveResultSizePolicy(Config{MaxResultBytes: 1 << 20}); got != 4096 {
		t.Fatalf("env>cfg: got %d, want 4096", got)
	}

	// 环境变量 <=0 ⇒ 关闭
	t.Setenv(MaxResultBytesEnv, "0")
	if got := ResolveResultSizePolicy(Config{MaxResultBytes: 1 << 20}); got != 0 {
		t.Fatalf("env=0: got %d, want 0 (disabled)", got)
	}

	t.Setenv(MaxResultBytesEnv, "-5")
	if got := ResolveResultSizePolicy(Config{MaxResultBytes: 1 << 20}); got != 0 {
		t.Fatalf("env<0: got %d, want 0 (disabled)", got)
	}

	// 非法环境变量 ⇒ 忽略，回退配置
	t.Setenv(MaxResultBytesEnv, "not-a-number")
	if got := ResolveResultSizePolicy(Config{MaxResultBytes: 8192}); got != 8192 {
		t.Fatalf("env invalid: got %d, want cfg 8192", got)
	}
}

func TestApplyResultSizePolicyRoundTrip(t *testing.T) {
	prev := MaxResultBytes()
	t.Cleanup(func() { SetMaxResultBytes(prev) })

	ApplyResultSizePolicy(Config{MaxResultBytes: 12345})
	if got := MaxResultBytes(); got != 12345 {
		t.Fatalf("applied limit = %d, want 12345", got)
	}

	ApplyResultSizePolicy(Config{MaxResultBytes: -1})
	if got := MaxResultBytes(); got != 0 {
		t.Fatalf("applied negative = %d, want 0 (disabled)", got)
	}
}

// 「上限 <=0 回退旧行为」：SetMaxResultBytes(<=0) 必须让上限变成 0（关闭），
// 而不是变成负数或保留旧值。
func TestSetMaxResultBytesNonPositiveDisables(t *testing.T) {
	prev := MaxResultBytes()
	t.Cleanup(func() { SetMaxResultBytes(prev) })

	for _, n := range []int64{0, -1, -1 << 40} {
		SetMaxResultBytes(n)
		if got := MaxResultBytes(); got != 0 {
			t.Fatalf("SetMaxResultBytes(%d): got %d, want 0", n, got)
		}
	}
}

func TestResultTooLargeErrorIsIdentifiable(t *testing.T) {
	err := &ResultTooLargeError{Op: "shard_query", Limit: 1024, Actual: 2048}

	if !IsResultTooLarge(err) {
		t.Fatalf("IsResultTooLarge(%T) = false, want true", err)
	}

	// 包装后仍可识别（errors.As 语义）。
	wrapped := fmt.Errorf("runShardQuery failed: %w", err)
	if !IsResultTooLarge(wrapped) {
		t.Fatalf("IsResultTooLarge(wrapped) = false, want true")
	}

	// 普通错误不可误判。
	if IsResultTooLarge(errors.New("boom")) {
		t.Fatalf("plain error misidentified as ResultTooLargeError")
	}

	// 错误消息必须能区分「太大」而不是「没数据」。
	msg := err.Error()
	if msg == "" {
		t.Fatalf("empty error message")
	}
	for _, want := range []string{"too large", "shard_query", "1024", "2048"} {
		if !contains(msg, want) {
			t.Fatalf("error message %q missing %q", msg, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// 两档上限：查询类 vs 元数据类
// ---------------------------------------------------------------------------

// 关键回归：元数据类 op 必须走**元数据档**，绝不能被查询档的 16 MiB 上限卡住。
// 第一版（单档）就是因为把启动必读的 segment_state 按 16 MiB 卡，导致聚合器
// 启动 exit 1 崩溃循环。
func TestLimitForOpRoutesMetadataToItsOwnTier(t *testing.T) {
	prevQ, prevM := MaxResultBytes(), MetadataMaxResultBytes()
	t.Cleanup(func() {
		SetMaxResultBytes(prevQ)
		SetMetadataMaxResultBytes(prevM)
	})

	SetMaxResultBytes(1000)
	SetMetadataMaxResultBytes(2000)

	metadataOps := []string{OpMetadata, OpSegmentState, "metadata:block_method_names"}
	for _, op := range metadataOps {
		if !IsMetadataOp(op) {
			t.Fatalf("IsMetadataOp(%q) = false, want true", op)
		}
		if got := LimitForOp(op); got != 2000 {
			t.Fatalf("LimitForOp(%q) = %d, want metadata limit 2000", op, got)
		}
	}

	queryOps := []string{"shard_query", "query", "boundary", "response:tipset", "test"}
	for _, op := range queryOps {
		if IsMetadataOp(op) {
			t.Fatalf("IsMetadataOp(%q) = true, want false", op)
		}
		if got := LimitForOp(op); got != 1000 {
			t.Fatalf("LimitForOp(%q) = %d, want query limit 1000", op, got)
		}
	}

	// 两档独立关闭：关掉元数据档不应影响查询档，反之亦然。
	SetMetadataMaxResultBytes(0)
	if got := LimitForOp(OpSegmentState); got != 0 {
		t.Fatalf("metadata tier disabled: LimitForOp(segment_state) = %d, want 0", got)
	}
	if got := LimitForOp("shard_query"); got != 1000 {
		t.Fatalf("metadata tier disabled must not affect query tier: got %d, want 1000", got)
	}
}

// 元数据档默认值必须**明显高于实测的启动元数据量级**（生产实测 16,777,354 字节），
// 否则启动又会因为元数据涨一点点就被拒。这里把「实测值」写进断言，防止有人把默认值调小。
func TestDefaultMetadataLimitCoversObservedStartupMetadata(t *testing.T) {
	// 2026-09-30 生产故障日志里的真实数字：
	// `op=segment_state, limit=16777216 bytes, exceeded at 16777354 bytes`
	const observedProductionMetadataBytes int64 = 16_777_354

	if DefaultMetadataMaxResultBytes <= observedProductionMetadataBytes {
		t.Fatalf("default metadata limit %d must exceed the observed startup metadata %d",
			DefaultMetadataMaxResultBytes, observedProductionMetadataBytes)
	}
	// 「明显高于」= 至少 8×（256 MiB / 16.0 MiB ≈ 16×，这里只锁住 8× 的下界）。
	if DefaultMetadataMaxResultBytes < 8*observedProductionMetadataBytes {
		t.Fatalf("default metadata limit %d must be >= 8x the observed startup metadata (%d); got only %.1fx",
			DefaultMetadataMaxResultBytes, observedProductionMetadataBytes,
			float64(DefaultMetadataMaxResultBytes)/float64(observedProductionMetadataBytes))
	}
	// 而且必须高于查询档 —— 元数据档比查询档还紧几乎肯定是配置/实现写反了。
	if DefaultMetadataMaxResultBytes <= DefaultMaxResultBytes {
		t.Fatalf("metadata default (%d) must be > query default (%d)",
			DefaultMetadataMaxResultBytes, DefaultMaxResultBytes)
	}
}

func TestResolveMetadataResultSizePolicyPrecedence(t *testing.T) {
	// 默认：配置文件零值 ⇒ 内置默认（256 MiB）
	if got := ResolveMetadataResultSizePolicy(Config{}); got != DefaultMetadataMaxResultBytes {
		t.Fatalf("zero config: got %d, want default %d", got, DefaultMetadataMaxResultBytes)
	}

	if got := ResolveMetadataResultSizePolicy(Config{MetadataMaxResultBytes: 1 << 30}); got != 1<<30 {
		t.Fatalf("cfg>0: got %d, want %d", got, 1<<30)
	}

	if got := ResolveMetadataResultSizePolicy(Config{MetadataMaxResultBytes: -1}); got != 0 {
		t.Fatalf("cfg<0: got %d, want 0 (disabled)", got)
	}

	// 环境变量优先于配置文件
	t.Setenv(MetadataMaxResultBytesEnv, "536870912")
	if got := ResolveMetadataResultSizePolicy(Config{MetadataMaxResultBytes: 1 << 30}); got != 536870912 {
		t.Fatalf("env>cfg: got %d, want 536870912", got)
	}

	t.Setenv(MetadataMaxResultBytesEnv, "0")
	if got := ResolveMetadataResultSizePolicy(Config{MetadataMaxResultBytes: 1 << 30}); got != 0 {
		t.Fatalf("env=0: got %d, want 0 (disabled)", got)
	}

	// 非法环境变量 ⇒ 忽略，回退配置
	t.Setenv(MetadataMaxResultBytesEnv, "nope")
	if got := ResolveMetadataResultSizePolicy(Config{MetadataMaxResultBytes: 4096}); got != 4096 {
		t.Fatalf("env invalid: got %d, want cfg 4096", got)
	}

	// 元数据档环境变量不得影响查询档（两档独立）。
	t.Setenv(MaxResultBytesEnv, "1234")
	p := ResolveResultSizePolicyFull(Config{})
	if p.QueryLimit != 1234 {
		t.Fatalf("query limit = %d, want 1234 (from %s)", p.QueryLimit, MaxResultBytesEnv)
	}
	if p.MetadataLimit != DefaultMetadataMaxResultBytes {
		t.Fatalf("metadata limit = %d, want default %d (env %s must not affect it)",
			p.MetadataLimit, DefaultMetadataMaxResultBytes, MaxResultBytesEnv)
	}
}

func TestResolveResultSizePolicyFullSources(t *testing.T) {
	// 全默认
	p := ResolveResultSizePolicyFull(Config{})
	if p.QuerySource != "default" || p.MetadataSource != "default" {
		t.Fatalf("sources = %q/%q, want default/default", p.QuerySource, p.MetadataSource)
	}
	if p.QueryLimit != DefaultMaxResultBytes || p.MetadataLimit != DefaultMetadataMaxResultBytes {
		t.Fatalf("limits = %d/%d, want %d/%d", p.QueryLimit, p.MetadataLimit,
			DefaultMaxResultBytes, DefaultMetadataMaxResultBytes)
	}

	// 配置文件来源
	p = ResolveResultSizePolicyFull(Config{MaxResultBytes: 1 << 20, MetadataMaxResultBytes: 1 << 28})
	if p.QuerySource != "config" || p.MetadataSource != "config" {
		t.Fatalf("sources = %q/%q, want config/config", p.QuerySource, p.MetadataSource)
	}

	// 配置文件显式关闭
	p = ResolveResultSizePolicyFull(Config{MaxResultBytes: -1, MetadataMaxResultBytes: -1})
	if p.QueryLimit != 0 || p.MetadataLimit != 0 || p.QuerySource != "config(disabled)" {
		t.Fatalf("disabled: %+v", p)
	}

	// 环境变量来源
	t.Setenv(MaxResultBytesEnv, "2048")
	p = ResolveResultSizePolicyFull(Config{MaxResultBytes: 1 << 20})
	if p.QueryLimit != 2048 || p.QuerySource != "env" {
		t.Fatalf("env source: %+v", p)
	}

	// String() 必须带上生效值与来源（日志可验证性）。
	s := p.String()
	for _, want := range []string{"query_limit=2048", "env", "metadata_limit="} {
		if !contains(s, want) {
			t.Fatalf("policy string %q missing %q", s, want)
		}
	}
}

func TestApplyResultSizePolicySetsBothTiers(t *testing.T) {
	prevQ, prevM := MaxResultBytes(), MetadataMaxResultBytes()
	t.Cleanup(func() {
		SetMaxResultBytes(prevQ)
		SetMetadataMaxResultBytes(prevM)
	})

	ApplyResultSizePolicy(Config{MaxResultBytes: 12345, MetadataMaxResultBytes: 98765})
	if got := MaxResultBytes(); got != 12345 {
		t.Fatalf("query tier = %d, want 12345", got)
	}
	if got := MetadataMaxResultBytes(); got != 98765 {
		t.Fatalf("metadata tier = %d, want 98765", got)
	}

	// 只写查询档 ⇒ 元数据档保持默认（配置文件里不写就是默认行为）。
	ApplyResultSizePolicy(Config{MaxResultBytes: 8192})
	if got := MaxResultBytes(); got != 8192 {
		t.Fatalf("query tier = %d, want 8192", got)
	}
	if got := MetadataMaxResultBytes(); got != DefaultMetadataMaxResultBytes {
		t.Fatalf("metadata tier = %d, want default %d", got, DefaultMetadataMaxResultBytes)
	}

	// 关闭元数据档
	ApplyResultSizePolicy(Config{MetadataMaxResultBytes: -1})
	if got := MetadataMaxResultBytes(); got != 0 {
		t.Fatalf("metadata tier = %d, want 0 (disabled)", got)
	}
}

// 「接近上限」告警：阈值可判定，且关闭上限时永不告警。
func TestNearLimitThreshold(t *testing.T) {
	const limit = 1000

	if NearLimit(0, limit) {
		t.Fatalf("used=0 must not be near limit")
	}
	if NearLimit(799, limit) {
		t.Fatalf("used=799/1000 (79%%) must not be near limit (threshold %v)", ResultSizeWarnRatio)
	}
	if !NearLimit(800, limit) {
		t.Fatalf("used=800/1000 (80%%) must be near limit")
	}
	if !NearLimit(1000, limit) {
		t.Fatalf("used=limit must be near limit")
	}

	// 关闭上限 ⇒ 永不告警（没有上限就没有「接近上限」这回事）。
	for _, l := range []int64{0, -1} {
		if NearLimit(1<<40, l) {
			t.Fatalf("limit=%d (disabled) must never warn", l)
		}
	}

	// 元数据档的实测量级（16.7 MB / 256 MiB ≈ 6.5%）绝不应噪声性触发。
	if NearLimit(16_777_354, DefaultMetadataMaxResultBytes) {
		t.Fatalf("observed startup metadata (16.7 MB) must not trigger the warn threshold at the default metadata limit")
	}
}
