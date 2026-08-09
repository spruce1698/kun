/**
 * @Author: albert
 * @Date: 2024-12-26
 * @Desc: csrf
 */

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
)

// CSRF 中间件
// 密钥应从配置或环境变量读取，禁止硬编码
// csrf.Protect 的 options 解析/令牌存储初始化成本属于一次性投入,在路由注册时执行一次,
// 之后仅按请求用同一中间件包装 handler(gorilla/csrf 支持并发复用)。
func CSRF(authKey []byte) gin.HandlerFunc {
	csrfProtect := csrf.Protect(
		authKey,
		csrf.Secure(false),               // 是否只在 HTTPS 中启用，开发阶段可以设置为 false
		csrf.RequestHeader("CSRF-Token"), // 通过header(CSRF-Token)获取 csrftoken
		// csrf.CookieName("CSRF-Token"), // 通过cookie(CSRF-Token)获取 csrftoken
		csrf.Path("/"),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.HttpOnly(true), // 指定cookie的值只能在服务端设置，禁止在客户端使用javascript修改
		// 校验失败时:写 403 响应;成功时 gorilla/csrf 不会调用本 handler,而是进入下面的内部 handler。
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`Forbidden - CSRF token invalid`))
		})),
	)

	return func(ctx *gin.Context) {
		// passed 标记 CSRF 校验是否通过:通过时被包装 handler 置位并推进 Gin 链;
		// 失败时由 ErrorHandler 写 403 响应,这里 Abort 截断链路,避免后续 handler(如 SubmitCsrf)
		// 重复写响应,出现两段 JSON 拼接的问题。
		passed := false

		// gorilla/csrf 默认按 TLS 模式做严格的 Origin/Referer 校验:非安全方法(POST 等)
		// 在没有 Origin 头时会强制要求 Referer。HTTP(明文)环境下客户端(如 Postman/内部服务)
		// 常不带 Referer,会被 ErrNoReferer 拦截。这里对真正明文请求标记为 plaintext,
		// 跳过 Referer 强校验,改为只校验 CSRF token 本身。
		//
		// TLS 判定同时考虑 X-Forwarded-Proto:TLS 终止代理(nginx https -> app http)下 r.TLS==nil,
		// 但请求实际是 HTTPS,应保留完整 Origin/Referer 校验,不能降级为 plaintext。
		req := ctx.Request
		if !isTLS(req) {
			req = csrf.PlaintextHTTPRequest(req)
		}

		csrfHandler := csrfProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.Request = r
			passed = true
		}))
		csrfHandler.ServeHTTP(ctx.Writer, req)

		if passed {
			ctx.Next()
			return
		}
		// 校验失败:gorilla/csrf 的 ErrorHandler 已写过 403,这里只需截断 Gin 链。
		ctx.Abort()
	}
}

// isTLS 判断请求是否经 TLS 传输。
// r.TLS != nil 表示直连 HTTPS;X-Forwarded-Proto=https 表示经 TLS 终止代理。
func isTLS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
