/**
 * @Author: spruce
 * @Date: 2024-03-28 11:11
 * @Desc: v0 路由
 */

package v0

import (
	"basic/internal/global"
	"basic/internal/handler"
	"basic/internal/middleware"
	"basic/pkg/token"

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

	}
	// mgr Demo路由
	mgrGroup := e.Group(global.RouterPrefixMgr).Use(middleware.Auth(jwt))
	{
		mgrGroup.GET("/demo/:id", ctx.DemoHandler.Detail)
	}
}
