package util

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	reward19 "github.com/filecoin-project/go-state-types/builtin/v19/reward"
	"github.com/robertkrimen/otto"
	"go.mongodb.org/mongo-driver/bson"
)

// 池化 VM 的全局作用域是共享的：连续调用之间不能串 ctx，
// 否则下一个请求会用上一个请求的参数查库（静默错数据，不报错）。
func TestParseCtxIsolation(t *testing.T) {
	src := `([{ $match: { Cid: ctx.Cid } }])`

	got, err := Parse(map[string]interface{}{"Cid": "bafyAAA"}, src)
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	if !containsValue(got, "bafyAAA") {
		t.Fatalf("first parse: want bafyAAA, got %v", got)
	}

	// 同一份源码、不同 ctx：脚本缓存不能把上一个 ctx 固化进去
	got, err = Parse(map[string]interface{}{"Cid": "bafyBBB"}, src)
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	if !containsValue(got, "bafyBBB") || containsValue(got, "bafyAAA") {
		t.Fatalf("second parse: got %v, want bafyBBB only", got)
	}

	// ctx 为 nil 时不能看到上一次的 ctx（归池时必须清干净）
	got, err = Parse(nil, `([{ $match: { Cid: (typeof ctx === "undefined" ? "no-ctx" : ctx.Cid) } }])`)
	if err != nil {
		t.Fatalf("third parse: %v", err)
	}
	if !containsValue(got, "no-ctx") {
		t.Fatalf("third parse: ctx leaked from previous call, got %v", got)
	}
}

// 本次改动的安全前提：编译产物只绑定 otto 运行时版本，不绑定 VM 实例。
// 这里显式用两个不同的 VM 验证「一个 VM 编译、另一个 VM 执行」成立。
func TestCompiledScriptReusableAcrossVMs(t *testing.T) {
	const src = `({ a: 1, b: "x" })`

	vm1 := otto.New()
	s1, err := compiledScript(vm1, src)
	if err != nil {
		t.Fatalf("compile with vm1: %v", err)
	}

	vm2 := otto.New()
	s2, err := compiledScript(vm2, src) // 命中缓存，返回 vm1 编译的那份
	if err != nil {
		t.Fatalf("compile with vm2: %v", err)
	}
	if s1 != s2 {
		t.Fatal("want cached script reused")
	}

	v, err := vm2.Run(s2)
	if err != nil {
		t.Fatalf("run cached script on another vm: %v", err)
	}
	agg, err := value2agg(v)
	if err != nil {
		t.Fatalf("value2agg: %v", err)
	}
	if !containsValue(agg, "x") {
		t.Fatalf("got %v, want field b=x", agg)
	}
}

// 编译失败的源码要报错，且不能污染缓存影响后续调用
func TestParseCompileError(t *testing.T) {
	if _, err := Parse(nil, `([{ $match: { `); err == nil {
		t.Fatal("want error for broken source")
	}

	if _, err := Parse(nil, `({ ok: true })`); err != nil {
		t.Fatalf("parse after broken source: %v", err)
	}
}

// 并发调用：池化与脚本缓存都必须并发安全，且不能串 ctx
func TestParseConcurrent(t *testing.T) {
	const src = `([{ $match: { Addr: ctx.Addr } }])`

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			want := fmt.Sprintf("addr-%d", i)
			got, err := Parse(map[string]interface{}{"Addr": want}, src)
			if err != nil {
				t.Errorf("parse: %v", err)
				return
			}
			if !containsValue(got, want) {
				t.Errorf("got %v, want %s", got, want)
			}
		}()
	}
	wg.Wait()
}

func containsValue(v interface{}, want string) bool {
	switch x := v.(type) {
	case bson.D:
		for _, e := range x {
			if containsValue(e.Value, want) {
				return true
			}
		}
	case []interface{}:
		for _, e := range x {
			if containsValue(e, want) {
				return true
			}
		}
	case string:
		return x == want
	}

	return false
}

// NV29(FIP-0118):奖励 actor 新增一组 FRC-0042 导出方法,方法名的唯一权威来源是
// go-state-types v0.19.0 的 builtin/v19/reward/methods.go(MethodMeta.Name)。
// AllMethodList 是方法筛选下拉与「按方法建桶」的唯一来源,缺一个名字就会让该方法
// 在筛选/聚合里漏掉。这里直接从依赖的 reward19.Methods 取名字断言,而不是在测试里
// 再抄一份硬编码清单,避免清单与上游两面漂移。
func TestAllMethodListCoversNV29RewardMethods(t *testing.T) {
	want := map[string]bool{}
	for _, meta := range reward19.Methods {
		if strings.HasSuffix(meta.Name, "Exported") {
			want[meta.Name] = true
		}
	}
	if len(want) == 0 {
		t.Fatal("reward19.Methods 未解析出任何 Exported 方法,依赖版本可能不是 v0.19.0")
	}

	have := make(map[string]bool, len(AllMethodList))
	for _, m := range AllMethodList {
		have[m] = true
	}

	for name := range want {
		if !have[name] {
			t.Errorf("AllMethodList 缺少 NV29 奖励流方法 %q", name)
		}
	}
}

// AllMethodList 是「按方法名建桶」的键集合:重名会让同一方法被建两次桶并双计;
// 既有约定是按 ASCII 升序维护列表,乱序说明插入时放错了位置。
func TestAllMethodListUniqueAndSorted(t *testing.T) {
	seen := make(map[string]bool, len(AllMethodList))
	for _, m := range AllMethodList {
		if seen[m] {
			t.Errorf("AllMethodList 含重复方法名 %q", m)
		}
		seen[m] = true
	}

	if !sort.StringsAreSorted(AllMethodList) {
		t.Error("AllMethodList 未按升序排列")
	}
}
