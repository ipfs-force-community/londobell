package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/lotus/node/config"
)

// 元数据缓存开关是新增的配置字段：必须确认
// (1) 写盘/读回后字段值不丢（配置巡检每 30s 会重写一次配置文件）；
// (2) 老配置文件（没有这两个字段）仍能解析，零值 = 开启缓存 + 默认 TTL。
func TestConfigMetaCacheFieldsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultConfig()
	cfg.Formal = NewDB("mongodb://formal:27017/?directConnection=true", "formal")
	cfg.Tmp = NewDB("mongodb://tmp:27017/?directConnection=true", "tmp")
	cfg.Colds = []DB{NewDB("mongodb://cold1:27017/?directConnection=true", "cold1")}
	cfg.DisableMetaCache = true
	cfg.MetaCacheTTLSeconds = 45

	if err := WriteToConfig(path, cfg); err != nil {
		t.Fatalf("WriteToConfig failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config file failed: %v", err)
	}

	if !strings.Contains(string(content), "DisableMetaCache") {
		t.Fatalf("config file does not contain DisableMetaCache:\n%s", content)
	}
	if !strings.Contains(string(content), "MetaCacheTTLSeconds") {
		t.Fatalf("config file does not contain MetaCacheTTLSeconds:\n%s", content)
	}

	// 读回：字段值必须原样恢复
	got := Config{}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open config file failed: %v", err)
	}
	defer file.Close()

	if _, err := config.FromReader(file, &got); err != nil {
		t.Fatalf("FromReader failed: %v", err)
	}

	if !got.DisableMetaCache || got.MetaCacheTTLSeconds != 45 {
		t.Fatalf("meta cache fields lost on reload: DisableMetaCache=%v, MetaCacheTTLSeconds=%v",
			got.DisableMetaCache, got.MetaCacheTTLSeconds)
	}
	if got.Formal.URL != cfg.Formal.URL || got.Tmp.DBName != cfg.Tmp.DBName || len(got.Colds) != 1 {
		t.Fatalf("existing fields changed: %+v", got)
	}

	// 老配置文件（把新字段整行删掉）必须仍能解析，且零值 = 默认行为（开启 + 默认 TTL）
	legacyLines := make([]string, 0, len(strings.Split(string(content), "\n")))
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "DisableMetaCache") || strings.Contains(line, "MetaCacheTTLSeconds") {
			continue
		}
		legacyLines = append(legacyLines, line)
	}

	legacyPath := filepath.Join(dir, "legacy.toml")
	if err := os.WriteFile(legacyPath, []byte(strings.Join(legacyLines, "\n")), 0644); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}

	legacy := Config{}
	legacyFile, err := os.Open(legacyPath)
	if err != nil {
		t.Fatalf("open legacy config failed: %v", err)
	}
	defer legacyFile.Close()

	if _, err := config.FromReader(legacyFile, &legacy); err != nil {
		t.Fatalf("FromReader for legacy config failed: %v", err)
	}
	if legacy.DisableMetaCache || legacy.MetaCacheTTLSeconds != 0 {
		t.Fatalf("zero value of meta cache fields must mean default: %+v", legacy)
	}
	if legacy.Formal.URL != cfg.Formal.URL {
		t.Fatalf("legacy config lost existing fields: %+v", legacy)
	}
}
