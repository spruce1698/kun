package xlog

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func BenchmarkFilterContent_JSON(b *testing.B) {
	jsonPayload := []byte(`{"id":123,"name":"john","role_id":1,"status":1,"created_at":"2026-08-24 12:00:00","items":[{"item_id":1,"count":10},{"item_id":2,"count":20}]}`)
	contentType := "application/json"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterContent(contentType, jsonPayload)
	}
}

func BenchmarkFilterContent_JSONWithSensitive(b *testing.B) {
	jsonPayload := []byte(`{"id":123,"name":"john","password":"mysecretpassword","token":"ey123456"}`)
	contentType := "application/json"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterContent(contentType, jsonPayload)
	}
}

func BenchmarkFilterHeaders(b *testing.B) {
	headers := http.Header{
		"Content-Type":    []string{"application/json"},
		"User-Agent":      []string{"Mozilla/5.0"},
		"Authorization":   []string{"Bearer eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9..."},
		"X-Forwarded-For": []string{"192.168.1.1"},
		"Accept":          []string{"application/json"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterHeaders(headers)
	}
}

func BenchmarkTracingWithLogger(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	eng := gin.New()
	logger := zap.NewNop()
	eng.Use(TracingWithLogger(logger, "test-bench"))
	eng.POST("/api/v1/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "data": "ok"})
	})

	body := []byte(`{"id":100,"name":"bench"}`)
	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req.Body = io.NopCloser(bytes.NewReader(body))
		eng.ServeHTTP(w, req)
	}
}
