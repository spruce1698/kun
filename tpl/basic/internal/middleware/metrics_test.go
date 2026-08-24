package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetrics_MiddlewareAndHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	engine.Use(Metrics("test-service"))
	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	engine.GET("/api/demo", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 请求业务路由
	req1 := httptest.NewRequest(http.MethodGet, "/api/demo", nil)
	w1 := httptest.NewRecorder()
	engine.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// 请求 /metrics 接口
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	wMetrics := httptest.NewRecorder()
	engine.ServeHTTP(wMetrics, reqMetrics)

	if wMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", wMetrics.Code)
	}

	body := wMetrics.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("expected /metrics to contain 'http_requests_total', got:\n%s", body)
	}
	if !strings.Contains(body, "http_request_duration_seconds") {
		t.Fatalf("expected /metrics to contain 'http_request_duration_seconds', got:\n%s", body)
	}
}
