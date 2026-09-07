/**
 * @Author: spruce
 * @Date: 2024-08-15
 * @Desc: Broker 事件订阅任务配置类型
 */

package event

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
