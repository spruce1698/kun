/**
 * @Author: spruce
 * @Date: 2024-08-15
 * @Desc: broker 事件订阅配置类型
 *
 * Task/Kafka/Asynq 定义了"订阅哪些 topic、由谁处理"的配置结构,
 * 被 pkg/xserver/broker(引擎)与 internal/event(消费者运行时)、
 * internal/service(装配)共同引用。放在 pkg 使 broker 引擎不再依赖 internal。
 * topic/group 使用原始 string:业务枚举(如 internal/global 的 EventTopic)由装配层转换。
 */

package xbroker

import (
	"context"

	"advanced/pkg/asynq"
)

// Task 订阅任务集合:一组 kafka 订阅 + 一组 asynq 订阅。
type Task struct {
	KafkaSet []*Kafka
	AsynqSet []*Asynq
}

// Kafka kafka 订阅项。
type Kafka struct {
	Topic   string
	Group   string
	Handler func(ctx context.Context, key, value string) error
}

// Asynq asynq 订阅项。
type Asynq struct {
	Topic   string
	Handler func(context.Context, *asynq.Task) error
}
