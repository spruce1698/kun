/**
 * @Author:
 * @Date: 2024-03-28 13:46
 * @Desc: 授权验证中间件
 */

package middleware

import (
	"strconv"

	"errors"

	"basic/internal/global"
	"basic/pkg/token"
	"basic/pkg/xconfig"
	"basic/pkg/xerror"
	"basic/pkg/xhttp"

	"github.com/gin-gonic/gin"
)

// Auth jwt授权
func Auth(jwt *token.Jwt) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		err := parsePayload(ctx, jwt)
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

// 可选授权
func OptAuth(jwt *token.Jwt) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_ = parsePayload(ctx, jwt)
		ctx.Next()
	}
}

// 如果有就写入授权
func WriteAuth(conf *xconfig.Conf, jwt *token.Jwt) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 查找token
		if tokenStr := jwt.Find(ctx.Request); tokenStr != "" {
			ctx.Set(global.CtxAuthToken, tokenStr)
		}
		ctx.Next()
	}
}

func parsePayload(ctx *gin.Context, jwt *token.Jwt) (err error) {
	// 查找token
	if tokenStr := jwt.Find(ctx.Request); tokenStr != "" {
		payload, parseErr := jwt.Parse(ctx.Request.Context(), tokenStr)
		if parseErr != nil {
			err = parseErr
		} else {
			userId, _ := strconv.ParseInt(payload.Subject, 10, 64)
			roleId, _ := strconv.ParseInt(payload.Issuer, 10, 64)
			ctx.Set(global.CtxAuthToken, tokenStr)
			ctx.Set(global.CtxAuthUserId, userId)
			ctx.Set(global.CtxAuthRoleId, roleId)
		}
	} else {
		err = token.ErrEmptyToken
	}
	return err
}
