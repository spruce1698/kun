package router

import (
	"basic/internal/handler"
	"basic/internal/router/v0"
	"basic/pkg/token"

	// ==== Add Rt import  before this line, don't edit this line.====
	"github.com/gin-gonic/gin"
)

// 定义路由注册接口
type (
	Router func(e *gin.Engine, jwt *token.Jwt, ctx *handler.Ctx)
)

// 路由层
func WireServerSet() []Router {
	return []Router{
		v0.Demo,
		// ==== Add Rt before this line, don't edit this line.====
	}
}
