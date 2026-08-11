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

	"advanced/pkg/xlog"

	"github.com/gin-gonic/gin"
)

// Recovery 项目可能出现的panic，并使用日志记录相关日志
func Recovery(stack bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		lCtx := ctx.Request.Context()
		defer func() {
			if err := recover(); err != nil {
				// 将任意 panic 值转为 error，避免非 error 类型导致二次 panic
				var panicErr error
				if e, ok := err.(error); ok {
					panicErr = e
				} else {
					panicErr = fmt.Errorf("%v", err)
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

				if stack {
					xlog.Error(lCtx, "[Recovery from panic]", panicErr, map[string]any{
						"request": string(httpRequest),
						"url":     ctx.Request.URL.Path,
						"stack":   string(debug.Stack()),
					})
				} else {
					xlog.Error(lCtx, "[Recovery from panic]", panicErr, map[string]any{
						"request": string(httpRequest),
						"url":     ctx.Request.URL.Path,
					})
				}
				// 返回统一 JSON 错误体,避免客户端收到空 500。
				// 若 header 已写出(tracing/handler 已写部分响应),WriteHeaderString 会告警但不崩。
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
					"code":    http.StatusInternalServerError,
					"message": "Internal Server Error",
				})
			}
		}()
		ctx.Next()
	}
}
