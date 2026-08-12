/**
 * @Author:
 * @Date: 2024-03-28 18:18
 * @Desc: 处理panic
 */

package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"basic/pkg/xlog"

	"github.com/gin-gonic/gin"
)

// Recovery 项目可能出现的panic，并使用日志记录相关日志
// verbose 仅控制是否把堆栈回显给客户端;堆栈始终写入日志。
func Recovery(verbose bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// panic 时重新取 context:下游中间件可能替换过 ctx.Request
				// (如 CSRF),用进入时缓存的 ctx 会丢掉 trace_id。
				lCtx := ctx.Request.Context()

				// 将任意 panic 值转为 error，避免非 error 类型导致二次 panic
				var panicErr error
				if e, ok := err.(error); ok {
					panicErr = e
				} else {
					panicErr = fmt.Errorf("%v", err)
				}

				// http.ErrAbortHandler 是 net/http 约定的"静默中止"信号
				// (如 ReverseProxy、WebSocket 劫持后写失败),不该打堆栈也不该写响应。
				if errors.Is(panicErr, http.ErrAbortHandler) {
					ctx.Abort()
					return
				}

				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				var ne *net.OpError
				if errors.As(panicErr, &ne) {
					var se *os.SyscallError
					if errors.As(ne.Err, &se) {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(ctx.Request, false)
				if brokenPipe {
					xlog.Error(lCtx, "Recovery HTTP request", panicErr, map[string]any{
						"request": string(httpRequest),
						"url":     ctx.Request.URL.Path,
					})
					// If the connection is dead, we can't write a status to it.
					_ = ctx.Error(panicErr)
					ctx.Abort()
					return
				}

				// 堆栈始终记录:生产环境没有堆栈的 panic 日志几乎无法定位问题。
				// verbose 只决定是否把堆栈也返回给客户端(绝不能在生产回显)。
				xlog.Error(lCtx, "[Recovery from panic]", panicErr, map[string]any{
					"request": string(httpRequest),
					"url":     ctx.Request.URL.Path,
					"stack":   string(debug.Stack()),
				})

				body := map[string]any{
					"code":    http.StatusInternalServerError,
					"message": "Internal Server Error",
				}
				if verbose {
					body["error"] = panicErr.Error()
					body["stack"] = string(debug.Stack())
				}
				// 返回统一 JSON 错误体,避免客户端收到空 500。
				// 若 header 已写出(tracing/handler 已写部分响应),WriteHeaderString 会告警但不崩。
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, body)
			}
		}()
		ctx.Next()
	}
}
