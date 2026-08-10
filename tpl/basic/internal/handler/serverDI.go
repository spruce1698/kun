package handler

import (
	"github.com/google/wire"
)

// 处理器列表
type Ctx struct {
	DemoHandler *DemoHandler
	// ==== Add Ctx before this line, don't edit this line.====
}

// 处理层
var WireServerSet = wire.NewSet(
	// 全部处理器
	wire.Struct(new(Ctx), "*"),

	// server 依赖的 handler
	wire.Struct(new(DemoHandler), "*"),
	// ==== Add Handler before this line, don't edit this line.====
)
