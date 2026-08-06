package service

import (
	"advanced/internal/event"
	"advanced/internal/service/svc"
	"advanced/pkg/token"

	"github.com/google/wire"
)

// 服务层
var WireServerSet = wire.NewSet(
	// 基础ctx
	wire.Struct(new(svc.Ctx), "*"),

	//  event
	event.NewPub,
	// ==== Add Event before this line, don't edit this line.====

	//  token (单例,与 http 层共用同一个 Jwt 实例)
	token.NewJwt,

	//  service
	wire.Struct(new(svc.DemoCtx), "*"),
	svc.NewDemoSvc,

	// ==== Add Svc before this line, don't edit this line.====
)
