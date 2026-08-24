/**
 * @Author: spruce
 * @Date: 2024-03-28 11:11
 * @Desc: v0 路由
 */

package v0

import (
	"advanced/internal/global"
	"advanced/internal/handler"
	"advanced/internal/middleware"
	"advanced/pkg/token"
	"time"

	"github.com/gin-gonic/gin"
)

func Demo(e *gin.Engine, jwt *token.Jwt, ctx *handler.Ctx) {
	// api Demo路由
	apiGroup := e.Group(global.RouterPrefixApi)
	{

		apiGroup.GET("/demo/:id", ctx.DemoHandler.Detail)
		apiGroup.GET("/demo/list", ctx.DemoHandler.List)
		apiGroup.POST("/demo/create", ctx.DemoHandler.Create)
		apiGroup.POST("/demo/update", ctx.DemoHandler.Update)
		apiGroup.POST("/demo/delete", ctx.DemoHandler.Delete)
		apiGroup.POST("/demo/softdelete", ctx.DemoHandler.SoftDelete)

		apiGroup.GET("/demo/validate", ctx.DemoHandler.Validate)

		apiGroup.GET("/demo/export", ctx.DemoHandler.Export)

		apiGroup.GET("/demo/excel", ctx.DemoHandler.Excel)

		apiGroup.GET("/demo/sse", ctx.DemoHandler.SSE)

		apiGroup.POST("/demo/sendMsg", ctx.DemoHandler.SendMsg)
		apiGroup.POST("/demo/addMsg", ctx.DemoHandler.AddMsg)
		apiGroup.POST("/demo/delMsg", ctx.DemoHandler.DelMsg)

		apiGroup.GET("/demo/csrf", middleware.CSRF([]byte(jwt.CSRFKey)), ctx.DemoHandler.GetCsrf)
		apiGroup.POST("/demo/submitCsrf", middleware.CSRF([]byte(jwt.CSRFKey)), ctx.DemoHandler.SubmitCsrf)

		// WebSocket
		apiGroup.GET("/demo/ws", ctx.DemoHandler.WS)

		// 登录相关(公开) — 限流: 每IP每分钟最多5次,超限封禁5分钟
		apiGroup.POST("/demo/login", middleware.RateLimiter(5, time.Minute, 5*time.Minute), ctx.DemoHandler.Login)
		apiGroup.POST("/demo/refresh", ctx.DemoHandler.Refresh)

		// 需授权路由
		authGroup := apiGroup.Group("/demo", middleware.Auth(jwt))
		{
			authGroup.GET("/info", ctx.DemoHandler.Info)
			authGroup.POST("/logout", ctx.DemoHandler.Logout)
			authGroup.POST("/ws-ticket", ctx.DemoHandler.WSTicket)
		}
	}
}
