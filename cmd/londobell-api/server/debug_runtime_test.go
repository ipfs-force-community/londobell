package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestDebugRuntimeRejectsPublicSource 公网来源必须被拒。
// 判定只看 RemoteAddr，因此伪造 X-Forwarded-For 也不能冒充内网。
func TestDebugRuntimeRejectsPublicSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/debug/runtime", DebugRuntime)

	req := httptest.NewRequest(http.MethodGet, "/debug/runtime", nil)
	req.RemoteAddr = "8.8.8.8:44321"
	req.Header.Set("X-Forwarded-For", "10.0.0.5") // 伪造也不该放行
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("公网来源状态码 = %d，期望 403", w.Code)
	}
}

// TestDebugRuntimeServesPrivateSource 私网来源正常返回，且带着排查需要的字段。
func TestDebugRuntimeServesPrivateSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/debug/runtime", DebugRuntime)

	req := httptest.NewRequest(http.MethodGet, "/debug/runtime", nil)
	req.RemoteAddr = "10.0.0.5:44321"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("私网来源状态码 = %d，期望 200", w.Code)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	for _, key := range []string{"goroutines", "stack_inuse_mb", "heap_inuse_mb", "shard_gate", "deadline_hits"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("缺少字段 %q（排障时靠它判断是否协程栈爆炸）", key)
		}
	}

	// 协程数至少是 1（当前测试协程）
	if g, _ := got["goroutines"].(float64); g < 1 {
		t.Fatalf("goroutines 读数异常: %v", got["goroutines"])
	}

	// 闸门快照必须带上限值，否则看门狗看不到当前生效策略
	gate, ok := got["shard_gate"].(map[string]interface{})
	if !ok {
		t.Fatalf("shard_gate 不是对象: %v", got["shard_gate"])
	}
	if _, ok := gate["limit"]; !ok {
		t.Fatal("shard_gate 缺少 limit 字段")
	}
}
