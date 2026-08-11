package kafka

import (
	"context"
	"sync"
	"time"

	"advanced/pkg/xconfig"

	"github.com/pkg/errors"
	kafkaGo "github.com/segmentio/kafka-go"
)

type (
	// Publisher 复用底层 kafka Writer,避免每次发布都新建/关闭 Writer 造成连接抖动和批处理失效。
	// kafka-go 的 Writer 并发安全,可被多个 goroutine 共享。
	Publisher struct {
		brokers []string
		writer  *kafkaGo.Writer
		once    sync.Once
	}
	Msg struct {
		Topic string
		Key   string
		Value []byte
	}
)

func NewPublisher(conf *xconfig.Conf) *Publisher {
	if len(conf.Kafka.Brokers) == 0 {
		panic("无可用的节点信息")
	}
	return &Publisher{
		brokers: conf.Kafka.Brokers,
	}
}

// getWriter 返回共享的 kafka Writer(懒初始化)。
// 不在 Writer 上设置 Topic,由每条消息自带 Topic 字段决定目标分区,从而支持 Pub/PubWithKey/PubList 共用一个 Writer。
func (p *Publisher) getWriter() *kafkaGo.Writer {
	p.once.Do(func() {
		p.writer = &kafkaGo.Writer{
			Addr:         kafkaGo.TCP(p.brokers...),
			Balancer:     &kafkaGo.LeastBytes{},
			Compression:  kafkaGo.Snappy,        // 启用压缩
			BatchTimeout: 50 * time.Millisecond, // linger.ms 如果消息的大小一直达不到batch.size设置的值，那么等待多久后任然允许发送消息
			// 配置为消息体积，而非条数，单位为字节
			Async: true, // 异步
		}
	})
	return p.writer
}

// Close 关闭底层 Writer,应在进程退出前调用。
func (p *Publisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

func (p *Publisher) Pub(topic string, value []byte) error {
	return p.PubCtx(context.Background(), topic, value)
}

// PubCtx 发布消息并把 ctx 中的 trace context 注入消息 header,串联生产者->消费者链路。
func (p *Publisher) PubCtx(ctx context.Context, topic string, value []byte) error {
	// 无类型消息:不设 Key(nil),由 LeastBytes balancer 均匀分配分区。
	ctx, span, headers := startProducerSpan(ctx, topic)
	defer span.End()
	msg := kafkaGo.Message{
		Topic:   topic,
		Value:   value,
		Headers: headers,
	}
	if err := p.getWriter().WriteMessages(ctx, msg); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (p *Publisher) PubWithKey(topic, key string, value []byte) error {
	return p.PubWithKeyCtx(context.Background(), topic, key, value)
}

// PubWithKeyCtx 同 PubCtx,但带业务 Key(决定分区)。
func (p *Publisher) PubWithKeyCtx(ctx context.Context, topic, key string, value []byte) error {
	ctx, span, headers := startProducerSpan(ctx, topic)
	defer span.End()
	msg := kafkaGo.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   value,
		Headers: headers,
	}
	if err := p.getWriter().WriteMessages(ctx, msg); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (p *Publisher) PubList(msgSet ...Msg) error {
	return p.PubListCtx(context.Background(), msgSet...)
}

// PubListCtx 批量发布,所有消息共享同一个 producer span。
func (p *Publisher) PubListCtx(ctx context.Context, msgSet ...Msg) error {
	if len(msgSet) == 0 {
		return errors.New("Msg 不能为空")
	}
	ctx, span, headers := startProducerSpan(ctx, "batch")
	defer span.End()
	msgList := make([]kafkaGo.Message, len(msgSet))
	for i, msg := range msgSet {
		msgList[i] = kafkaGo.Message{
			Topic:   msg.Topic,
			Key:     []byte(msg.Key),
			Value:   msg.Value,
			Headers: headers,
		}
	}
	if err := p.getWriter().WriteMessages(ctx, msgList...); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
