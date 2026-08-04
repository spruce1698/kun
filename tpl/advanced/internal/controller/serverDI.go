package controller

import (
	"advanced/internal/controller/demo"

	"github.com/google/wire"
)

// 控制器列表
type ServerCtrlCtx struct {
	DemoCtrl *demo.DemoCtrl
	// ==== Add CtrlCtx before this line, don't edit this line.====
}

// 控制层
var WireServerSet = wire.NewSet(
	// 全部控制器
	wire.Struct(new(ServerCtrlCtx), "*"),

	// server 依赖的 controller
	wire.Struct(new(demo.DemoCtrl), "*"),
	// ==== Add Ctrl before this line, don't edit this line.====
)
