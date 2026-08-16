//go:build wireinject
// +build wireinject

package wire

import (
	"basic/internal/app"
	"basic/internal/handler"
	"basic/internal/repository"
	"basic/internal/repository/db"
	"basic/internal/router"
	"basic/internal/service"
	"basic/pkg/token"
	"basic/pkg/xconfig"
	"basic/pkg/xdb"
	"basic/pkg/xlog"
	"basic/pkg/xredis"
	"basic/pkg/xserver"
	xhttp "basic/pkg/xserver/http"

	"github.com/google/wire"
)

// 依赖注入构造 Web 应用声明函数
func WireApp(env string) (*xserver.Server, error) {
	panic(wire.Build(
		// ====base,配置,日志,数据库链接等====
		xconfig.New,
		xlog.New,

		xdb.New,
		xredis.New,

		db.NewConn,

		// token 单例:http 装配层(WriteAuth/路由)与 service 层共用同一实例,
		// 保证黑名单/轮换全局生效。
		token.NewJwt,

		// app 装配 gin 引擎 + 资源 closer
		app.NewHttp,
		wire.FieldsOf(new(*app.Assembly), "Engine", "Closer"),

		xhttp.New,

		repository.WireServerSet,
		service.WireServerSet,
		handler.WireServerSet,
		router.WireServerSet,

		xserver.New,
	))
}
