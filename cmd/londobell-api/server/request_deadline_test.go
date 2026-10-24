package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestRequestDeadlineWrites504 超时且 handler 没写过响应时，中间件必须补 504。
// 这条用例对应事故里的现象：客户端放弃后服务端还在跑，最后什么也没返回。
func TestRequestDeadlineWrites504(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("LONDOBELL_REQUEST_TIMEOUT", "80ms")

	r := gin.New()
	r.Use(RequestDeadline())
	r.POST("/slow", func(c *gin.Context) {
		<-c.Request.Context().Done() // 模拟一个卡在分片查询里的 handler
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/slow", nil)

	start := time.Now()
	r.ServeHTTP(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("状态码 = %d，期望 %d", w.Code, http.StatusGatewayTimeout)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("中间件没有按 timeout 生效，耗时 %s", elapsed)
	}
}

// TestRequestDeadlineKeepsHandlerResponse 超时前就写完响应的请求，中间件不得覆盖它的响应。
func TestRequestDeadlineKeepsHandlerResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("LONDOBELL_REQUEST_TIMEOUT", "1s")

	r := gin.New()
	r.Use(RequestDeadline())
	r.POST("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/ok", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200（中间件不该改动已完成的响应）", w.Code)
	}
	if body := w.Body.String(); body != `{"code":0}` {
		t.Fatalf("响应体被改写: %s", body)
	}
}

// TestRequestDeadlinePropagatesClientCancel 客户端断开必须让 handler 的 ctx 立刻结束。
// 这是本补丁的核心：换成 c.Request.Context() 之后，服务端不再替已离开的客户端干活。
func TestRequestDeadlinePropagatesClientCancel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("LONDOBELL_REQUEST_TIMEOUT", "30s") // 故意大于测试时长，证明是「客户端取消」而不是超时

	var cancelled int32
	var wg sync.WaitGroup
	wg.Add(1)

	r := gin.New()
	r.Use(RequestDeadline())
	r.POST("/wait", func(c *gin.Context) {
		defer wg.Done()
		<-c.Request.Context().Done()
		atomic.StoreInt32(&cancelled, 1)
	})

	req := httptest.NewRequest(http.MethodPost, "/wait", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel() // 客户端断开
	}()

	r.ServeHTTP(httptest.NewRecorder(), req)
	wg.Wait()

	if atomic.LoadInt32(&cancelled) != 1 {
		t.Fatal("handler 的 ctx 没有被客户端断开取消")
	}
}

// TestRequestDeadlineCanBeDisabled LONDOBELL_REQUEST_TIMEOUT=0 时只继承客户端取消，不加 deadline。
func TestRequestDeadlineCanBeDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("LONDOBELL_REQUEST_TIMEOUT", "0")

	var hasDeadline bool
	r := gin.New()
	r.Use(RequestDeadline())
	r.POST("/x", func(c *gin.Context) {
		_, hasDeadline = c.Request.Context().Deadline()
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x", nil))
	if hasDeadline {
		t.Fatal("timeout=0 时不应设置 deadline")
	}
}
