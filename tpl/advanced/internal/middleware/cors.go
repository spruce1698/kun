/**
 * @Author:
 * @Date: 2024-03-28 10:41
 * @Desc: 跨域中间件
 */

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
// allowOrigins 为白名单列表;为空时允许所有来源(开发环境)
func CORS(allowOrigins ...[]string) gin.HandlerFunc {
	var origins map[string]struct{}
	if len(allowOrigins) > 0 && len(allowOrigins[0]) > 0 {
		origins = make(map[string]struct{}, len(allowOrigins[0]))
		for _, o := range allowOrigins[0] {
			origins[strings.ToLower(o)] = struct{}{}
		}
	}

	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		// 白名单校验
		if origins != nil && origin != "*" {
			if _, ok := origins[strings.ToLower(origin)]; !ok {
				ctx.String(http.StatusForbidden, "CORS: origin %q not allowed", origin)
				ctx.Abort()
				return
			}
		}

		// 允许的header类型
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With,X-Request-Id,Client-Type,App-Version,Token")
		// 跨域允许的请求方式
		ctx.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		// 跨域允许的请求时间(秒)
		ctx.Header("Access-Control-Max-Age", "86400")

		if origin == "*" {
			// 不带凭证时可以用 *
			ctx.Header("Access-Control-Allow-Origin", "*")
		} else {
			// 带凭证时必须回显具体 Origin
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
