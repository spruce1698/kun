/**
 * @Author: spruce
 * @Date: 2024-03-28 11:08
 * @Desc: 路由层
 */

package router

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"basic/pkg/xhttp"
	"basic/pkg/xserver"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 缺省路由
func NotFoundHandle(ctx *gin.Context) {
	xhttp.WithNotFoundPath(ctx)
}

// Metrics 暴露 Prometheus 抓取端点
func Metrics(e *gin.Engine) {
	e.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// CheckFunc 探活函数定义
type CheckFunc func(ctx context.Context) error

// NamedChecker 具名健康检测项
type NamedChecker struct {
	Name  string
	Check CheckFunc
}

// HealthChecks 暴露云原生健康检查探针端点:
// - /healthz: 存活探针 (Liveness Probe)，进程存活且端口响应即返回 UP
// - /ready:   就绪探针 (Readiness Probe)，依赖基础设施(如 DB/Redis)全部就绪且未处于停机排空状态
// 支持 GET 与 HEAD 请求方法，并发带超时检测并支持 panic 兜底。
func HealthChecks(e *gin.Engine, checkers ...NamedChecker) {
	e.Match([]string{http.MethodGet, http.MethodHead}, "/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	e.Match([]string{http.MethodGet, http.MethodHead}, "/ready", func(ctx *gin.Context) {
		// 1. 服务若处于优雅下线排空阶段，立即返回 503 摘除流量
		if xserver.IsDraining() {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "DRAINING",
				"message": "service is shutting down",
			})
			return
		}

		if len(checkers) == 0 {
			ctx.JSON(http.StatusOK, gin.H{"status": "READY"})
			return
		}

		// 2. 限制单次就绪检查总超时为 2 秒，防止下游阻塞挂死探针
		checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			hasError bool
			results  = make(map[string]any, len(checkers))
		)

		for _, item := range checkers {
			if item.Check == nil {
				continue
			}
			wg.Add(1)
			go func(c NamedChecker) {
				defer wg.Done()

				var checkErr error
				defer func() {
					if r := recover(); r != nil {
						checkErr = fmt.Errorf("panic: %v", r)
					}
					mu.Lock()
					defer mu.Unlock()
					if checkErr != nil {
						hasError = true
						results[c.Name] = gin.H{
							"status": "DOWN",
							"error":  checkErr.Error(),
						}
					} else {
						results[c.Name] = gin.H{
							"status": "UP",
						}
					}
				}()

				checkErr = c.Check(checkCtx)
			}(item)
		}

		wg.Wait()

		status := "READY"
		code := http.StatusOK
		if hasError {
			status = "UNAVAILABLE"
			code = http.StatusServiceUnavailable
		}

		ctx.JSON(code, gin.H{
			"status": status,
			"checks": results,
		})
	})
}
