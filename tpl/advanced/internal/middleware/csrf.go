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
func CSRF(authKey []byte) gin.HandlerFunc {
	csrfProtection := csrf.Protect(
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
		// 常不带 Referer,会被 ErrNoReferer 拦截。这里对非 TLS 请求标记为 plaintext,
		// 跳过 Referer 强校验,改为只校验 CSRF token 本身。
		// 生产 HTTPS 环境下 r.TLS != nil,不会进入此分支,仍保留完整校验。
		req := ctx.Request
		if req.TLS == nil {
			req = csrf.PlaintextHTTPRequest(req)
		}

		csrfHandler := csrfProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
