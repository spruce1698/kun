/**
 * @Author: spruce
 * @Date: 2024-03-28 11:08
 * @Desc: 路由层
 */

package router

import (
	"net/http"

	"basic/pkg/xhttp"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 缺省路由
func NotFoundHandle(ctx *gin.Context) {
	xhttp.WithNotFoundPath(ctx)
}

// ping 路由
func Ping(e *gin.Engine) {
	e.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})
}

// Metrics 暴露 Prometheus 抓取端点
func Metrics(e *gin.Engine) {
	e.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
