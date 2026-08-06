package service

import (
	"advanced/internal/event"
	"advanced/internal/global"
	"advanced/internal/service/svc"

	"github.com/google/wire"
)

// 订阅服务
func TaskSet(svc svc.BrokerSvc) *event.Task {
	return &event.Task{
		KafkaSet: []*event.Kafka{
			{
				Topic:   global.EventTopicPay,
				Group:   global.EventGroupPay,
				Handler: svc.SubPayRecharge,
			},
			{
				Topic:   global.EventTopicPay,
				Group:   global.EventGroupPay1,
				Handler: svc.KQPayDemo,
			},
		},
		AsynqSet: []*event.Asynq{
			{
				Topic:   global.EventTopicPay,
				Handler: svc.SubAqPay,
			},
			{
				Topic:   global.EventTopicPay1,
				Handler: svc.SubAqPay1,
			},
		},
	}
}

// 服务层
var WireBrokerSet = wire.NewSet(
	// 基础ctx
	wire.Struct(new(svc.Ctx), "*"),

	// service
	wire.Struct(new(svc.BrokerCtx), "*"),
	svc.NewBrokerSvc,
	// ==== Add Svc before this line, don't edit this line.====

	// 订阅服务
	TaskSet,
)
