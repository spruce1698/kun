/**
 * @Author:
 * @Date: 2024-03-28 13:46
 * @Desc: 授权验证中间件
 */

package middleware

import (
	"errors"
	"strconv"

	"basic/internal/global"
	"basic/pkg/token"
	"basic/pkg/xerror"
	"basic/pkg/xhttp"

	"github.com/gin-gonic/gin"
)

// Auth jwt授权
func Auth(jwt *token.Jwt) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, exists := ctx.Get(global.CtxAuthUserId); exists {
			ctx.Next()
			return
		}
		tokenStr := jwt.Find(ctx.Request)
		err := parsePayload(ctx, jwt, tokenStr)
		if err != nil {
			var e *token.Error
			if errors.As(err, &e) {
				switch e.Errors {
				case token.ErrorExpiredToken:
					xhttp.AuthFail(ctx, xerror.AuthTimeOut)
					ctx.Abort()
					return
				default:
					xhttp.AuthFail(ctx, xerror.AuthError)
					ctx.Abort()
					return
				}
			}
			xhttp.AuthFail(ctx, xerror.AuthError)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// OptAuth 可选授权
func OptAuth(jwt *token.Jwt) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, exists := ctx.Get(global.CtxAuthUserId); !exists {
			tokenStr := jwt.Find(ctx.Request)
			_ = parsePayload(ctx, jwt, tokenStr)
		}
		ctx.Next()
	}
}

// WriteAuth 全局可选解析——有 token 就写入 context,没有也放行。
// conf 参数已移除(无实际用途),调用方签名简化为 WriteAuth(jwt)。
func WriteAuth(jwt *token.Jwt) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if tokenStr := jwt.Find(ctx.Request); tokenStr != "" {
			_ = parsePayload(ctx, jwt, tokenStr)
		}
		ctx.Next()
	}
}

// parsePayload 解析 JWT 并将 userId/roleId 写入 gin context。
// tokenStr 由调用方预先通过 jwt.Find 获取，避免在同一请求中重复解析 Header/Query。
func parsePayload(ctx *gin.Context, jwt *token.Jwt, tokenStr string) (err error) {
	if tokenStr == "" {
		return token.ErrEmptyToken
	}
	payload, parseErr := jwt.Parse(ctx.Request.Context(), tokenStr)
	if parseErr != nil {
		return parseErr
	}
	userId, _ := strconv.ParseInt(payload.Subject, 10, 64)
	roleId, _ := strconv.ParseInt(payload.Issuer, 10, 64)
	ctx.Set(global.CtxAuthToken, tokenStr)
	ctx.Set(global.CtxAuthUserId, userId)
	ctx.Set(global.CtxAuthRoleId, roleId)
	return nil
}
