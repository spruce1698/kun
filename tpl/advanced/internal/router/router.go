/**
 * @Author: spruce
 * @Date: 2024-03-28 11:08
 * @Desc: 路由层 wire 配置
 */

package router

import (
	"net/http"

	"advanced/pkg/xhttp"
	"advanced/swagger"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NotFoundHandle 缺省路由
func NotFoundHandle(ctx *gin.Context) {
	xhttp.WithNotFoundPath(ctx)
}

// Ping 路由
func Ping(e *gin.Engine) {
	e.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})
}

// SwaggerRouter 路由
func SwaggerRouter(e *gin.Engine) {
	swagger.SwaggerInfo.BasePath = ""
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // /swagger/index.html
}
