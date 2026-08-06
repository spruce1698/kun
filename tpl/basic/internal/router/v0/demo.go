/**
 * @Author: spruce
 * @Date: 2024-03-28 11:11
 * @Desc: v0 路由
 */

package v0

import (
	"basic/internal/controller"
	"basic/internal/global"
	"basic/internal/middleware"
	"basic/pkg/token"

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

	}
	// mgr Demo路由
	mgrGroup := e.Group(global.RouterPrefixMgr).Use(middleware.Auth(jwt))
	{
		mgrGroup.GET("/demo", ctx.DemoCtrl.Detail)
	}
}
