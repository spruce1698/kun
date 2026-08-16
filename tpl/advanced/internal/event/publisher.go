/**
* @Author: spruce
 * @Date: 2024-03-28 17:27
 * @Desc: 消息推送器
*/

package event

import (
	"context"
	"fmt"
	"time"

	"advanced/internal/global"
	"advanced/pkg/asynq"
	"advanced/pkg/kafka"
	"advanced/pkg/xconfig"
	"advanced/pkg/xlog"
)

type (
	Pub struct {
		kQueue *kafka.Publisher
		aQueue *asynq.Asynq
		logger *xlog.Logger
	}
)

func NewPub(conf *xconfig.Conf, log *xlog.Logger) *Pub {
	return &Pub{
		kQueue: kafka.NewPublisher(conf),
		aQueue: asynq.New(conf),
		logger: log,
	}
}

// Close 关闭底层 kafka Writer 等资源,应在进程退出前调用。
func (p *Pub) Close() {
	if err := p.kQueue.Close(); err != nil {
		p.logger.Warn(fmt.Sprintf("kafka publisher close err: %v", err))
	}
}

// Kafka 发布kafka异步消息
func (p *Pub) Kafka(topic global.EventTopic, message []byte) error {
	return p.KafkaCtx(context.Background(), topic, message)
}

// KafkaCtx 发布kafka异步消息,并把 ctx 中的 trace context 注入消息 header,
// 串联 HTTP(producer)->kafka->consumer 跨进程链路。
func (p *Pub) KafkaCtx(ctx context.Context, topic global.EventTopic, message []byte) error {
	topicStr := string(topic)
	err := p.kQueue.PubCtx(ctx, topicStr, message)
	if err != nil {
		return fmt.Errorf("kafka topic:%s 发布消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// KafkaWithType 发布kafka异步有type的消息
func (p *Pub) KafkaWithType(topic global.EventTopic, msgType global.EventType, message []byte) error {
	return p.KafkaWithTypeCtx(context.Background(), topic, msgType, message)
}

// KafkaWithTypeCtx 同 KafkaWithType,带 ctx 以传播 trace context。
func (p *Pub) KafkaWithTypeCtx(ctx context.Context, topic global.EventTopic, msgType global.EventType, message []byte) error {
	topicStr := string(topic)
	msgTypeStr := string(msgType)
	err := p.kQueue.PubWithKeyCtx(ctx, topicStr, msgTypeStr, message)
	if err != nil {
		return fmt.Errorf("kafka topic:%s 发布类型:%s 消息:%s 失败, error:%w", topicStr, msgTypeStr, message, err)
	}
	return nil
}

// Delay 发布延时消息,单位s
func (p *Pub) Delay(topic global.EventTopic, message string, delay int64) error {
	return p.DelayCtx(context.Background(), topic, message, delay)
}

// DelayCtx 同 Delay,带 ctx 以在 trace 中记录入队 span。
func (p *Pub) DelayCtx(ctx context.Context, topic global.EventTopic, message string, delay int64) error {
	topicStr := string(topic)
	_, err := p.aQueue.DelayPubCtx(ctx, asynq.CriticalQueue, topicStr, message, time.Duration(delay)*time.Second)
	if err != nil {
		return fmt.Errorf("topic:%s  发布延时消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// Sync 发布异步消息
func (p *Pub) Sync(topic global.EventTopic, message string) error {
	return p.SyncCtx(context.Background(), topic, message)
}

// SyncCtx 同 Sync,带 ctx 以在 trace 中记录入队 span。
func (p *Pub) SyncCtx(ctx context.Context, topic global.EventTopic, message string) error {
	topicStr := string(topic)
	_, err := p.aQueue.SyncPubCtx(ctx, asynq.DefaultQueue, topicStr, message)
	if err != nil {
		return fmt.Errorf("topic:%s  发布异步消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// Cron 发布定时消息
func (p *Pub) Cron(topic global.EventTopic, message, cronSpec string) error {
	topicStr := string(topic)
	err := p.aQueue.SetCronPub(topicStr, message, cronSpec)
	if err != nil {
		return fmt.Errorf("topic:%s  发布周期/定时消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// DelCron 发布取消消息
func (p *Pub) DelCron(topic global.EventTopic) error {
	topicStr := string(topic)
	err := p.aQueue.DelCronPub(topicStr)
	if err != nil {
		return fmt.Errorf("topic:%s  取消周期/定时消息 失败, error:%w", topicStr, err)
	}
	return nil
}
