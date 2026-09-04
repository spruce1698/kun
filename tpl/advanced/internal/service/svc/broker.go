/**
 * @Author: albert
 * @Date: 2024-07-02
 * @Desc: broker Svc
 */

package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"advanced/internal/global"
	"advanced/pkg/asynq"
	"advanced/pkg/utils"
	"advanced/pkg/xlog"
)

//go:generate mockgen -source=./broker.go -destination=../../../test/mocks/service/broker.go  -package mock_service

var _ BrokerSvc = (*brokerSvc)(nil)

type (
	BrokerSvc interface {
		SubPayRecharge(ctx context.Context, key, value string) error
		KQPayDemo(ctx context.Context, key, value string) error
		SubAqPay(ctx context.Context, task *asynq.Task) error
		SubAqPay1(ctx context.Context, task *asynq.Task) error
	}
	BrokerCtx struct {
		*Ctx
	}
	brokerSvc struct {
		ctx *BrokerCtx
	}
)

func NewBrokerSvc(ctx *BrokerCtx) BrokerSvc {
	return &brokerSvc{
		ctx: ctx,
	}
}

// SubPayRecharge 支付事件订阅
func (b *brokerSvc) SubPayRecharge(ctx context.Context, key, value string) error {
	switch global.EventType(key) {
	case global.EventTypePayRecharge: // 充值
		msg := &global.PayRecharge{}
		if err := json.Unmarshal([]byte(value), msg); err != nil {
			xlog.Error(ctx, "SubPayRecharge 反序列化失败", err, value)
			return err
		}

		xlog.Info(ctx, fmt.Sprintf("SubPayRecharge-type:消费时间:%s:%s", utils.TimeStr(time.Now()), value))

		// TODO: do something 逻辑处理start...

		return nil
	default:
		xlog.Error(ctx, "未知的支付事件类型:"+key+value, nil)
	}
	return nil
}

// KQPayDemo 支付事件订阅,无类型
func (b *brokerSvc) KQPayDemo(ctx context.Context, key, value string) error {
	msg := &global.PayRecharge{}
	if err := json.Unmarshal([]byte(value), msg); err != nil {
		xlog.Error(ctx, "KQPayDemo 反序列化失败", err, value)
		return err
	}

	xlog.Info(ctx, fmt.Sprintf("KQPayDemo:not key 消费时间:%s:%s", utils.TimeStr(time.Now()), value))
	xlog.Info(ctx, b.ctx.Conf.Get().Env+" 测试:"+key+value)

	// TODO: do something 逻辑处理start...

	return nil
}

// SubAqPay aq订阅
func (b *brokerSvc) SubAqPay(ctx context.Context, task *asynq.Task) error {
	xlog.Info(ctx, fmt.Sprintf("SubAqPay:消费时间:%s: %s %s", utils.TimeStr(time.Now()), task.Type(), task.Payload()))

	// TODO: do something 逻辑处理start...

	return nil
}

func (b *brokerSvc) SubAqPay1(ctx context.Context, task *asynq.Task) error {
	xlog.Info(ctx, fmt.Sprintf("SubAqPay1:消费时间:%s: %s %s", utils.TimeStr(time.Now()), task.Type(), task.Payload()))

	// TODO: do something 逻辑处理start...
	return nil
}
