package kafka

import (
	"context"
	"sync"
	"time"

	"advanced/pkg/xconfig"

	"github.com/pkg/errors"
	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
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
			// BatchSize		batch.size	当多条消息被发送到同一个分区时，生产者会尝试把多条消息变成批量发送。这有助于提高客户端和服务器的性能。值设置的太小，可能会降低吞吐量 参数设置的太大，可能会更浪费内存，并增加消息发送的延迟时间
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

func (p *Publisher) Pub(ctx context.Context, topic string, value []byte) error {
	// 无类型消息:不设 Key(nil),由 LeastBytes balancer 均匀分配分区,
	// 避免用 nano 时间戳做 Key 造成同毫秒消息挤在同一分区且无业务含义。
	msg := kafkaGo.Message{
		Topic: topic,
		Value: value,
	}
	// 注入 trace context 到消息 headers,供消费者端提取建立链路
	otel.GetTextMapPropagator().Inject(ctx, &kafkaHeadersCarrier{headers: &msg.Headers})
	return p.getWriter().WriteMessages(ctx, msg)
}

func (p *Publisher) PubWithKey(ctx context.Context, topic, key string, value []byte) error {
	msg := kafkaGo.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	otel.GetTextMapPropagator().Inject(ctx, &kafkaHeadersCarrier{headers: &msg.Headers})
	return p.getWriter().WriteMessages(ctx, msg)
}

func (p *Publisher) PubList(ctx context.Context, msgSet ...Msg) error {
	if len(msgSet) == 0 {
		return errors.New("Msg 不能为空")
	}
	msgList := make([]kafkaGo.Message, len(msgSet))
	for i, msg := range msgSet {
		msgList[i] = kafkaGo.Message{
			Topic: msg.Topic,
			Key:   []byte(msg.Key),
			Value: msg.Value,
		}
		// 注入 trace context 到每条消息 headers
		otel.GetTextMapPropagator().Inject(ctx, &kafkaHeadersCarrier{headers: &msgList[i].Headers})
	}
	return p.getWriter().WriteMessages(ctx, msgList...)
}
