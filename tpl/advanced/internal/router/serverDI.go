package router

import (
	"advanced/internal/controller"
	"advanced/internal/router/v0"
	"advanced/pkg/token"

	// ==== Add Rt import  before this line, don't edit this line.====
	"github.com/gin-gonic/gin"
)

// 定义路由注册接口
type (
	Router func(e *gin.Engine, jwt *token.Jwt, ctx *controller.ServerCtrlCtx)
)

// 路由层
func WireServerSet() []Router {
	return []Router{
		v0.Demo,
		// ==== Add Rt before this line, don't edit this line.====
	}
}
