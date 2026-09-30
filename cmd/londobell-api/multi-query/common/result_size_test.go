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
