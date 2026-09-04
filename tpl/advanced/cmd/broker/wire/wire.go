//go:build wireinject
// +build wireinject

package wire

import (
	"advanced/internal/repository/db"
	"advanced/internal/service"
	"advanced/pkg/xconfig"
	"advanced/pkg/xdb"
	"advanced/pkg/xlog"
	"advanced/pkg/xredis"
	"advanced/pkg/xserver"
	"advanced/pkg/xserver/broker"

	"github.com/google/wire"
)

func WireApp(env string) (*xserver.Server, error) {
	panic(wire.Build(
		// ====base,配置,日志,数据库链接等====
		xconfig.New,
		xlog.New,

		xdb.New,
		xredis.New,

		db.NewConn,

		// broker 父进程健康探针 gin 引擎与子进程入口装配
		NewBrokerHealth,
		NewBrokerChildRun,

		broker.New,

		service.WireBrokerSet,

		xserver.New,
	))
}
