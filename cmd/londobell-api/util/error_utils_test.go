package util

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
)

func newRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// 「结果集超上限」必须走 HTTP 5xx，且错误体里能识别出这是「太大」而不是「没数据」。
func TestReturnOnErrMapsResultTooLargeToHTTP5xx(t *testing.T) {
	prev := common.MaxResultBytes()
	t.Cleanup(func() { common.SetMaxResultBytes(prev) })

	c, w := newRecorder()
	err := &common.ResultTooLargeError{Op: "shard_query", Limit: 1024, Actual: 4096}
	ReturnOnErr(c, err)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (5xx)", w.Code, http.StatusInternalServerError)
	}
	if w.Code < 500 || w.Code > 599 {
		t.Fatalf("status = %d, want a 5xx", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "too large") {
		t.Fatalf("body %q should be identifiable as a size-limit error", body)
	}
	if !strings.Contains(body, "shard_query") {
		t.Fatalf("body %q should carry the op label", body)
	}

	// 包装后同样要 5xx（errors.As 语义）。
	c2, w2 := newRecorder()
	ReturnOnErr(c2, errors.New("wrapped: "+err.Error()))
	if w2.Code != http.StatusOK {
		// 纯字符串包装无法被 errors.As 识别，这里只确认「非 ResultTooLarge」仍是旧行为（200）。
		t.Fatalf("plain wrapped string: status = %d, want legacy 200", w2.Code)
	}

	c3, w3 := newRecorder()
	var wrapped error = err
	for i := 0; i < 3; i++ {
		wrapped = wrapErr(wrapped)
	}
	ReturnOnErr(c3, wrapped)
	if w3.Code != http.StatusInternalServerError {
		t.Fatalf("wrapped ResultTooLargeError: status = %d, want 500", w3.Code)
	}
}

// 普通业务错误保持旧行为（HTTP 200 + code=Fail / NotFound），不能误伤。
func TestReturnOnErrKeepsLegacyBehaviorForOtherErrors(t *testing.T) {
	c, w := newRecorder()
	ReturnOnErr(c, errors.New("boom"))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want legacy 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "boom") {
		t.Fatalf("body %q should contain the error message", w.Body.String())
	}

	c2, w2 := newRecorder()
	ReturnOnErr(c2, ErrNotFound)
	if w2.Code != http.StatusOK {
		t.Fatalf("ErrNotFound status = %d, want legacy 200", w2.Code)
	}
}

func TestMarshalBoundedNormalAndOverLimit(t *testing.T) {
	prev := common.MaxResultBytes()
	t.Cleanup(func() { common.SetMaxResultBytes(prev) })

	type payload struct {
		A int      `json:"a"`
		B []string `json:"b"`
	}
	v := payload{A: 1, B: []string{"x", "y"}}

	// 正常：与 json.Marshal 逐字节一致。
	common.SetMaxResultBytes(common.DefaultMaxResultBytes)
	got, err := MarshalBounded(v, "response:test")
	if err != nil {
		t.Fatalf("under limit: unexpected error %v", err)
	}
	if string(got) != `{"a":1,"b":["x","y"]}` {
		t.Fatalf("under limit: got %s", got)
	}

	// 超限：显式错误，绝不返回被截断的 JSON。
	common.SetMaxResultBytes(4)
	got, err = MarshalBounded(v, "response:test")
	if err == nil {
		t.Fatalf("over limit: want error, got %s", got)
	}
	if got != nil {
		t.Fatalf("over limit: must NOT return (possibly truncated) bytes, got %s", got)
	}
	if !common.IsResultTooLarge(err) {
		t.Fatalf("over limit: want ResultTooLargeError, got %T: %v", err, err)
	}

	// 上限关闭（<=0）⇒ 回退旧行为。
	common.SetMaxResultBytes(0)
	got, err = MarshalBounded(v, "response:test")
	if err != nil {
		t.Fatalf("disabled: unexpected error %v", err)
	}
	if string(got) != `{"a":1,"b":["x","y"]}` {
		t.Fatalf("disabled: got %s", got)
	}
}

type wrapError struct{ err error }

func (w wrapError) Error() string { return "wrap: " + w.err.Error() }
func (w wrapError) Unwrap() error { return w.err }

func wrapErr(err error) error { return wrapError{err: err} }
