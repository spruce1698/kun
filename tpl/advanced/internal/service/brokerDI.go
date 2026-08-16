package service

import (
	"advanced/internal/global"
	"advanced/internal/service/svc"
	"advanced/pkg/xbroker"
	// ==== Add Svc import before this line, don't edit this line.====

	"github.com/google/wire"
)

// 订阅服务
func TaskSet(svc svc.BrokerSvc) *xbroker.Task {
	return &xbroker.Task{
		KafkaSet: []*xbroker.Kafka{
			{
				Topic:   string(global.EventTopicPay),
				Group:   string(global.EventGroupPay),
				Handler: svc.SubPayRecharge,
			},
			{
				Topic:   string(global.EventTopicPay),
				Group:   string(global.EventGroupPay1),
				Handler: svc.KQPayDemo,
			},
		},
		AsynqSet: []*xbroker.Asynq{
			{
				Topic:   string(global.EventTopicPay),
				Handler: svc.SubAqPay,
			},
			{
				Topic:   string(global.EventTopicPay1),
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
