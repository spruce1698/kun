//go:build wireinject
// +build wireinject

package wire

import (
	"advanced/internal/handler"
	"advanced/internal/repository"
	"advanced/internal/repository/db"
	"advanced/internal/router"
	"advanced/internal/service"
	"advanced/pkg/xconfig"
	"advanced/pkg/xdb"
	"advanced/pkg/xlog"
	"advanced/pkg/xredis"
	"advanced/pkg/xserver"
	xhttp "advanced/pkg/xserver/http"
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

		// app 装配 gin 引擎 + 资源 closer;jwt 仍由 service 层的 token.NewJwt 单例注入,
		// NewHttp 与 service 共用同一实例。
		NewHttp,
		wire.FieldsOf(new(*Assembly), "Engine", "Closer"),

		xhttp.New,

		repository.WireServerSet,
		service.WireServerSet,
		handler.WireServerSet,
		router.WireServerSet,

		xserver.New,
	))
}
