package service

import (
	"basic/internal/service/svc"

	"github.com/google/wire"
)

// 服务层
var WireServerSet = wire.NewSet(
	// 基础ctx
	wire.Struct(new(svc.Ctx), "*"),

	//  service
	wire.Struct(new(svc.DemoCtx), "*"),
	svc.NewDemoSvc,

	// ==== Add Svc before this line, don't edit this line.====
)
