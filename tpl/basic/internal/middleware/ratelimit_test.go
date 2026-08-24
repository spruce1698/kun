package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_InMemory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// 允许 2 次，窗口 1 秒，封禁 1 秒
	engine.Use(RateLimiter(2, time.Second, time.Second))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 第一次：200
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()
	engine.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// 第二次：200
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w2 := httptest.NewRecorder()
	engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	// 第三次：超限 429
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w3 := httptest.NewRecorder()
	engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w3.Code)
	}
}

func TestRedisRateLimiter_NilFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// rdb 为 nil 时降级放行
	engine.Use(RedisRateLimiter(nil, "ratelimit:test:", 2, time.Second))
	engine.GET("/test-nil", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-nil", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when redis is nil, got %d", w.Code)
	}
}
