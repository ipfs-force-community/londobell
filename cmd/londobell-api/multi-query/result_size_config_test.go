package multiquery

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/lotus/node/config"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
)

// ---------------------------------------------------------------------------
// ② 旋钮可验证：**配置文件**（common.Config.MaxResultBytes / MetadataMaxResultBytes）
// 这条路径必须真生效，运维不必依赖环境变量（上一版生产灰度就是 supervisor 的
// environment= 没进到进程，limit 仍然是内置默认 16777216）。
//
// 这里走的每一步都与生产热重载完全一致：
//	写配置文件        : common.WriteToConfig（multiQueryCfg 命令同款）
//	解码              : config.FromReader   （MonitorConfig 里同款）
//	应用              : dbsm.SetConfig      （MonitorConfig 里同款 ⇒ ApplyResultSizePolicy）
// 并断言「生效值 + 来源」被明确打进日志。
// ---------------------------------------------------------------------------

// captureSubsystemLogs 把指定子系统的日志 tee 一份到内存 buffer，返回该 buffer。
// 只提高该子系统的日志级别（go-log 默认 LevelError），避免影响其他测试的输出。
func captureSubsystemLogs(t *testing.T, subsystem string) *bytes.Buffer {
	t.Helper()

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encCfg)

	buf := &bytes.Buffer{}

	// tee 到 stderr，避免这个测试「吃掉」日志（便于排查）。
	logging.SetPrimaryCore(zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel),
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), zapcore.DebugLevel),
	))
	if err := logging.SetLogLevel(subsystem, "debug"); err != nil {
		t.Fatalf("set log level for %q: %v", subsystem, err)
	}

	t.Cleanup(func() {
		_ = logging.SetLogLevel(subsystem, "error")
		logging.SetPrimaryCore(zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), zapcore.DebugLevel))
	})

	return buf
}

func TestConfigFileKnobAppliesResultSizeLimitsAndLogsEffectiveValues(t *testing.T) {
	// 环境变量必须为空：否则无法证明「配置文件」这条路径本身生效。
	t.Setenv(common.MaxResultBytesEnv, "")
	t.Setenv(common.MetadataMaxResultBytesEnv, "")

	prevQ, prevM := common.MaxResultBytes(), common.MetadataMaxResultBytes()
	t.Cleanup(func() {
		common.SetMaxResultBytes(prevQ)
		common.SetMetadataMaxResultBytes(prevM)
	})

	logs := captureSubsystemLogs(t, "multi-query/result-size")

	// SetConfig 会顺带重配闸门；保持默认（并在测试结束时还原）。
	useGate(t, DefaultShardQueryConcurrency, DefaultShardQueryWait)

	const (
		wantQueryLimit    = int64(3 << 20)   // 3 MiB
		wantMetadataLimit = int64(384 << 20) // 384 MiB
	)

	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := common.DefaultConfig()
	cfg.MaxResultBytes = wantQueryLimit
	cfg.MetadataMaxResultBytes = wantMetadataLimit

	if err := common.WriteToConfig(cfgPath, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	for _, want := range []string{"MaxResultBytes", "MetadataMaxResultBytes"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("config file does not contain knob %q:\n%s", want, raw)
		}
	}

	// 解码：与 MonitorConfig 完全相同的路径。
	f, err := os.Open(cfgPath)
	if err != nil {
		t.Fatalf("open config file: %v", err)
	}
	defer f.Close()

	decoded := common.Config{}
	if _, err := config.FromReader(f, &decoded); err != nil {
		t.Fatalf("config.FromReader (hot-reload decode path): %v", err)
	}
	if decoded.MaxResultBytes != wantQueryLimit {
		t.Fatalf("decoded MaxResultBytes = %d, want %d", decoded.MaxResultBytes, wantQueryLimit)
	}
	if decoded.MetadataMaxResultBytes != wantMetadataLimit {
		t.Fatalf("decoded MetadataMaxResultBytes = %d, want %d", decoded.MetadataMaxResultBytes, wantMetadataLimit)
	}

	// 应用：与 MonitorConfig 完全相同的路径（SetConfig → ApplyResultSizePolicy）。
	dbsm := &DataBaseStateManager{DBCfg: common.NewDBCollectionsConfigMgr(common.Config{})}
	dbsm.SetConfig(decoded)

	if got := common.MaxResultBytes(); got != wantQueryLimit {
		t.Fatalf("applied query limit = %d, want %d (config-file knob must take effect without env vars)", got, wantQueryLimit)
	}
	if got := common.MetadataMaxResultBytes(); got != wantMetadataLimit {
		t.Fatalf("applied metadata limit = %d, want %d (config-file knob must take effect without env vars)", got, wantMetadataLimit)
	}

	// 路由：元数据 op 走元数据档，其余走查询档。
	if got := common.LimitForOp(common.OpSegmentState); got != wantMetadataLimit {
		t.Fatalf("LimitForOp(segment_state) = %d, want metadata limit %d", got, wantMetadataLimit)
	}
	if got := common.LimitForOp(common.OpMetadata); got != wantMetadataLimit {
		t.Fatalf("LimitForOp(metadata) = %d, want metadata limit %d", got, wantMetadataLimit)
	}
	if got := common.LimitForOp("shard_query"); got != wantQueryLimit {
		t.Fatalf("LimitForOp(shard_query) = %d, want query limit %d", got, wantQueryLimit)
	}

	// 日志：生效值 + 来源必须可读可 grep。
	out := logs.String()
	t.Logf("policy log line: %s", strings.TrimSpace(out))
	for _, want := range []string{
		"result size policy applied",
		"query_limit=3145728",      // 3 MiB
		"metadata_limit=402653184", // 384 MiB
		"config",                   // 来源
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("applied-policy log line missing %q; got:\n%s", want, out)
		}
	}

	// 关闭语义同样经配置文件生效：<0 = 关闭该档（回退旧行为）。
	cfg.MaxResultBytes = -1
	cfg.MetadataMaxResultBytes = -1
	if err := common.WriteToConfig(cfgPath, cfg); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}
	f2, err := os.Open(cfgPath)
	if err != nil {
		t.Fatalf("reopen config: %v", err)
	}
	defer f2.Close()

	decoded2 := common.Config{}
	if _, err := config.FromReader(f2, &decoded2); err != nil {
		t.Fatalf("FromReader #2: %v", err)
	}
	dbsm.SetConfig(decoded2)
	if common.MaxResultBytes() != 0 || common.MetadataMaxResultBytes() != 0 {
		t.Fatalf("negative config must disable both tiers, got query=%d metadata=%d",
			common.MaxResultBytes(), common.MetadataMaxResultBytes())
	}
}
