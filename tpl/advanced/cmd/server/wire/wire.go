//go:build wireinject
// +build wireinject

package wire

import (
	"advanced/internal/controller"
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

		xhttp.New,

		repository.WireServerSet,
		service.WireServerSet,
		controller.WireServerSet,
		router.WireServerSet,

		xserver.New,
	))
}
