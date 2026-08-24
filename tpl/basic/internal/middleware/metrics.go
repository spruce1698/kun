package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by status code, method, and HTTP path.",
		},
		[]string{"service", "method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds, partitioned by status code, method, and HTTP path.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed.",
		},
		[]string{"service"},
	)
)

// Metrics Prometheus HTTP 指标收集中间件
func Metrics(serviceName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 跳过 metrics 自身抓取和探针路由的耗时统计,避免指标自增循环
		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		if path == "/metrics" || path == "/ping" {
			ctx.Next()
			return
		}

		httpRequestsInFlight.WithLabelValues(serviceName).Inc()
		start := time.Now()

		ctx.Next()

		duration := time.Since(start).Seconds()
		httpRequestsInFlight.WithLabelValues(serviceName).Dec()

		statusStr := strconv.Itoa(ctx.Writer.Status())
		httpRequestsTotal.WithLabelValues(serviceName, ctx.Request.Method, path, statusStr).Inc()
		httpRequestDuration.WithLabelValues(serviceName, ctx.Request.Method, path, statusStr).Observe(duration)
	}
}
