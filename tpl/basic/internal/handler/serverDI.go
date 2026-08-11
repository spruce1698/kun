package handler

import (
	// ==== Add Handler import before this line, don't edit this line.====
	"github.com/google/wire"
)

// Ctx 处理器列表
type Ctx struct {
	DemoHandler *DemoHandler
	// ==== Add Handler to Ctx before this line, don't edit this line.====
}

// WireServerSet 处理层
var WireServerSet = wire.NewSet(
	// 全部处理器
	wire.Struct(new(Ctx), "*"),

	// server 依赖的 handler
	wire.Struct(new(DemoHandler), "*"),
	// ==== Add Handler to WireSet before this line, don't edit this line.====
)
