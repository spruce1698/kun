package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func BenchmarkCORS(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	eng := gin.New()
	eng.Use(CORS([]string{"https://app.example.com", "https://admin.example.com", "https://mall.example.com"}))
	eng.GET("/api/v1/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Origin", "https://app.example.com")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, req)
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	eng := gin.New()
	eng.Use(RateLimiter(1000000, time.Minute, time.Minute))
	eng.GET("/api/v1/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, req)
	}
}

func BenchmarkMetrics(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	eng := gin.New()
	eng.Use(Metrics("bench-svc"))
	eng.GET("/api/v1/users/:id", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, req)
	}
}
