//go:build wireinject
// +build wireinject

package wire

import (
	"basic/internal/handler"
	"basic/internal/repository"
	"basic/internal/repository/db"
	"basic/internal/router"
	"basic/internal/service"
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

		xhttp.New,

		repository.WireSet,
		service.WireSet,
		handler.WireSet,
		router.WireSet,

		xserver.New,
	))
}
