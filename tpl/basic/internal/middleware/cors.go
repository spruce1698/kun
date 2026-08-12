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
//
// allowOrigins 为来源白名单:
//   - 含具体来源:仅白名单内的 Origin 被放行,回显该 Origin 并允许携带凭证。
//   - 含 "*":显式允许所有来源。按规范此时只能下发 Allow-Origin: *,
//     且不能下发 Allow-Credentials(浏览器会拒绝 * 与凭证的组合),
//     故该模式仅适用于无需 Cookie/Authorization 的公开只读接口。
//   - 为空:视为未配置跨域,不下发任何 CORS 头,跨域请求由浏览器同源策略拦截。
//
// 安全要点:白名单为空时绝不能反射请求里的 Origin —— 反射 Origin 同时下发
// Access-Control-Allow-Credentials: true,等于允许任意站点带着用户 Cookie
// 读取本服务的响应(完整的 CORS 绕过)。
func CORS(allowOrigins ...[]string) gin.HandlerFunc {
	var (
		origins      map[string]struct{}
		allowAny     bool
		hasWhitelist bool
	)
	if len(allowOrigins) > 0 && len(allowOrigins[0]) > 0 {
		origins = make(map[string]struct{}, len(allowOrigins[0]))
		for _, o := range allowOrigins[0] {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			// "*" 只能作为配置项显式声明通配,不接受来自请求头的 "*"。
			if o == "*" {
				allowAny = true
				continue
			}
			origins[strings.ToLower(o)] = struct{}{}
			hasWhitelist = true
		}
	}

	return func(ctx *gin.Context) {
		// Origin 影响响应内容,必须声明 Vary,否则 CDN/反向代理可能把某个 Origin 的
		// Access-Control-Allow-Origin 缓存后回给另一个 Origin(跨源缓存投毒)。
		ctx.Header("Vary", "Origin")

		origin := ctx.Request.Header.Get("Origin")
		// 无 Origin 头:同源请求或非浏览器客户端(curl/服务端调用),无需下发 CORS 头。
		if origin == "" {
			ctx.Next()
			return
		}

		// 校验来源。注意:origin 来自请求头,即使其字面量为 "*" 也必须走白名单校验,
		// 不能当作通配处理(否则客户端只要发 `Origin: *` 就能绕过白名单)。
		allowed := allowAny
		if !allowed && hasWhitelist {
			_, allowed = origins[strings.ToLower(origin)]
		}
		if !allowed {
			ctx.String(http.StatusForbidden, "CORS: origin %q not allowed", origin)
			ctx.Abort()
			return
		}

		// 允许的header类型
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With,X-Request-Id,Client-Type,App-Version,Token")
		// 跨域允许的请求方式
		ctx.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		// 跨域允许的请求时间(秒)
		ctx.Header("Access-Control-Max-Age", "86400")

		if allowAny {
			// 通配模式:不能与 Allow-Credentials 同时下发。
			ctx.Header("Access-Control-Allow-Origin", "*")
		} else {
			// 白名单命中:回显具体 Origin,允许携带凭证。
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
