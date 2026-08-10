package handler

import (
	"advanced/internal/handler/demo"

	"github.com/google/wire"
)

// 处理器列表
type ServerHandlerCtx struct {
	DemoHandler *demo.DemoHandler
	// ==== Add HandlerCtx before this line, don't edit this line.====
}

// 处理层
var WireServerSet = wire.NewSet(
	// 全部处理器
	wire.Struct(new(ServerHandlerCtx), "*"),

	// server 依赖的 handler
	wire.Struct(new(demo.DemoHandler), "*"),
	// ==== Add Handler before this line, don't edit this line.====
)
