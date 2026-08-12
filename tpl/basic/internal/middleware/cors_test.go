package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func runCORS(t *testing.T, allow []string, reqOrigin string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(allow))
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if reqOrigin != "" {
		req.Header.Set("Origin", reqOrigin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 白名单为空时不得反射任意 Origin,更不得下发 Allow-Credentials。
func TestCORS_EmptyWhitelistDoesNotReflect(t *testing.T) {
	w := runCORS(t, nil, "https://evil.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("empty whitelist must not emit Allow-Origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("empty whitelist must not emit Allow-Credentials, got %q", got)
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", w.Code)
	}
}

// 客户端发送字面量 `Origin: *` 不得绕过白名单。
func TestCORS_LiteralStarOriginCannotBypass(t *testing.T) {
	w := runCORS(t, []string{"http://127.0.0.1:9998"}, "*")
	if w.Code != http.StatusForbidden {
		t.Fatalf("literal `Origin: *` must be rejected, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("must not emit Allow-Origin for literal star, got %q", got)
	}
}

// 白名单命中:回显具体 Origin 并允许凭证。
func TestCORS_WhitelistedOriginEchoed(t *testing.T) {
	w := runCORS(t, []string{"http://127.0.0.1:9998"}, "http://127.0.0.1:9998")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:9998" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials allowed, got %q", got)
	}
}

// 配置显式写 "*":下发通配但绝不同时下发 Allow-Credentials。
func TestCORS_ExplicitWildcardHasNoCredentials(t *testing.T) {
	w := runCORS(t, []string{"*"}, "https://anything.example")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard must never be combined with credentials, got %q", got)
	}
}

// 必须始终声明 Vary: Origin,避免跨源缓存投毒。
func TestCORS_VaryOriginAlwaysSet(t *testing.T) {
	w := runCORS(t, []string{"http://127.0.0.1:9998"}, "http://127.0.0.1:9998")
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary: Origin, got %q", got)
	}
}

// 无 Origin 头(同源/curl)应正常放行且不下发 CORS 头。
func TestCORS_NoOriginPassesThrough(t *testing.T) {
	w := runCORS(t, []string{"http://127.0.0.1:9998"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("request without Origin must pass, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("must not emit Allow-Origin without Origin, got %q", got)
	}
}
