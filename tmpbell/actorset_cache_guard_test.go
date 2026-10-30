package tmpbell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestForkHandlingMustNotClearActorSet 守住「分叉不清理 actor set 缓存」这条结论。
//
// 为什么必须有这条机检：actor set 缓存里是主网 772 万条（count=3855373 + init-state=3871211）
// actor code 映射，清一次就要全量重建，实测 4.5~6.5 分钟；这段时间 tmp 库最新 tipset
// 不前进，也就是「实时水位」停更。主网分叉极密（2026-10-30 实测 11 小时 18 次），
// 一旦在分叉处理里再调 ClearActorSet()，每天都白停 90 分钟，且与常规的预算法重建
// 撞在同一高度时会叠成 12 分钟停更，直接触发「filscan数据超过5/10分钟未更新」告警。
//
// 不复用缓存的正确性由两处兜底，不需要在分叉处清缓存：
//  1. acquireActorSet 的有界失效：网络版本变化 / 30 轮复用预算 / epoch 窗口；
//  2. LookupMethodInfo 的逐消息兜底：缓存未命中的 actor 走 stm.LoadActor 从该 tipset
//     状态读取并回填缓存。
func TestForkHandlingMustNotClearActorSet(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("没有找到包内源文件，机检失效（本测试必须在 tmpbell 包目录内运行）")
	}

	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(src), "ClearActorSet") {
			t.Fatalf("%s 出现 ClearActorSet：分叉/常规路径清空 actor set 缓存会让下一次抽取"+
				"全量重建 772 万条映射（实测 4.5~6.5 分钟水位停更）。正确性已由"+
				"acquireActorSet 的有界失效与 LookupMethodInfo 的 LoadActor 兜底覆盖，"+
				"如确需清缓存请先改这条机检并在 PR 里说明代价。", name)
		}
	}
}
