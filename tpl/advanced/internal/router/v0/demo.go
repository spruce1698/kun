/**
 * @Author: spruce
 * @Date: 2024-03-28 11:11
 * @Desc: v0 路由
 */

package v0

import (
	"advanced/internal/controller"
	"advanced/internal/global"
	"advanced/internal/middleware"
	"advanced/pkg/token"
	"time"

	"github.com/gin-gonic/gin"
)

func Demo(e *gin.Engine, jwt *token.Jwt, ctx *controller.ServerCtrlCtx) {
	// api Demo路由
	apiGroup := e.Group(global.RouterPrefixApi)
	{
		apiGroup.GET("/demo/:id", ctx.DemoCtrl.Detail)
		apiGroup.GET("/demo/list", ctx.DemoCtrl.List)

		apiGroup.POST("/demo/create", ctx.DemoCtrl.Create)
		apiGroup.POST("/demo/update", ctx.DemoCtrl.Update)
		apiGroup.POST("/demo/delete", ctx.DemoCtrl.Delete)
		apiGroup.POST("/demo/softdelete", ctx.DemoCtrl.SoftDelete)

		apiGroup.GET("/demo/export", ctx.DemoCtrl.Export)
		apiGroup.GET("/demo/excel", ctx.DemoCtrl.Excel)
		apiGroup.GET("/demo/sse", ctx.DemoCtrl.SSE)

		apiGroup.POST("/demo/sendMsg", ctx.DemoCtrl.SendMsg)
		apiGroup.POST("/demo/addMsg", ctx.DemoCtrl.AddMsg)
		apiGroup.POST("/demo/delMsg", ctx.DemoCtrl.DelMsg)

		apiGroup.GET("/demo/csrf", middleware.CSRF([]byte(jwt.CSRFKey)), ctx.DemoCtrl.GetCsrf)
		apiGroup.POST("/demo/submitCsrf", middleware.CSRF([]byte(jwt.CSRFKey)), ctx.DemoCtrl.SubmitCsrf)

		// WebSocket
		apiGroup.GET("/demo/ws", ctx.DemoCtrl.WS)

		// 登录相关(公开) — 限流: 每IP每分钟最多5次,超限封禁5分钟
		apiGroup.POST("/demo/login", middleware.RateLimiter(5, time.Minute, 5*time.Minute), ctx.DemoCtrl.Login)
		apiGroup.POST("/demo/refresh", ctx.DemoCtrl.Refresh)

		// 需授权路由
		authGroup := apiGroup.Group("/demo", middleware.Auth(jwt))
		{
			authGroup.GET("/info", ctx.DemoCtrl.Info)
			authGroup.POST("/logout", ctx.DemoCtrl.Logout)
			authGroup.POST("/ws-ticket", ctx.DemoCtrl.WSTicket)
		}
	}
}
