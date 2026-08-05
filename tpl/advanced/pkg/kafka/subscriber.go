package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"advanced/pkg/xconfig"

	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

const (
	topicGroupKey = "%s:%s"
	// maxConsumeRetry 单条消息处理失败时的最大重试次数。
	// 超过后视为毒消息,跳过提交(避免无限重投),仅记录日志。
	maxConsumeRetry = 3
)

// kafkaHeadersCarrier 实现 textmap.TextMapCarrier 接口,将 trace context 读写到 kafka 消息 headers。
// 使 OTel 的 Inject/Extract 能从 kafka 消息传播 trace_id/span_id。
type kafkaHeadersCarrier struct {
	headers *[]kafkaGo.Header
}

func (c *kafkaHeadersCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeadersCarrier) Set(key, value string) {
	// 追加而非覆盖,避免 OTel 标准头(traceparent/tracestate)被重复写入
	*c.headers = append(*c.headers, kafkaGo.Header{Key: key, Value: []byte(value)})
}

func (c *kafkaHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		keys = append(keys, h.Key)
	}
	return keys
}

type (
	Subscriber struct {
		brokers []string
		box     sync.Map
	}
)

func NewSubscriber(conf *xconfig.Conf) *Subscriber {
	if len(conf.Kafka.Brokers) == 0 {
		panic("无可用的节点信息")
	}
	return &Subscriber{
		brokers: conf.Kafka.Brokers,
		box:     sync.Map{},
	}
}

// Sub 订阅者 kafka提交 ,ctx 取消时退出消费循环。
// 同一 topic:group 仅创建一个 reader 并缓存,Close 时统一关闭。
func (s *Subscriber) Sub(ctx context.Context, topic, group string, handler func(ctx context.Context, key, value string) error) {
	reader := s.getOrCreateReader(topic, group, kafkaGo.ReaderConfig{
		StartOffset: kafkaGo.FirstOffset,
		Brokers:     s.brokers,
		GroupID:     group,
		Topic:       topic,
		MinBytes:    10e3, // 10KB fetch.min.bytes 服务器应该为获取请求返回的最小数据量。如果没有足够的数据可用，请求将等待那么多数据累积后再响应请求
		MaxBytes:    10e6, // 10MB
		// CommitInterval: time.Second,           // 提交间隔指示将偏移提交到代理的间隔。如果为 0，则同步处理提交。 要加 GroupID 才能自动提交
		MaxWait: time.Millisecond, // fetch.max.wait.ms 默认10s如果没有足够的数据立即满足fetch.min.bytes提供的要求，服务器在响应fetch请求之前将阻塞的最长时间
	})
	for {
		msg, err := reader.ReadMessage(ctx)
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			break
		}
		if err != nil {
			fmt.Printf("--sub-: %v\n", err)
			break
		}

		// 从消息 headers 提取 trace context,创建 child span
		msgCtx := otel.GetTextMapPropagator().Extract(context.Background(), &kafkaHeadersCarrier{headers: &msg.Headers})
		tracer := otel.Tracer("kafka-consumer")
		msgCtx, span := tracer.Start(msgCtx, fmt.Sprintf("consume %s", topic))
		// 处理失败时按指数退避重试,超过 maxConsumeRetry 仍失败则跳过该消息(kafka 自动提交会推进 offset),
		// 避免handler持续报错导致消费卡死。
		consumeWithRetry(ctx, string(msg.Key), string(msg.Value), func(_ context.Context, k, v string) error {
			return handler(msgCtx, k, v)
		})
		span.End()
	}
}

// SubFetch 订阅者 程序提交(速度快), ctx 取消时退出消费循环。
// 同一 topic:group 仅创建一个 reader 并缓存,Close 时统一关闭。
func (s *Subscriber) SubFetch(ctx context.Context, topic, group string, handler func(ctx context.Context, key, value string) error) {
	reader := s.getOrCreateReader(topic, group, kafkaGo.ReaderConfig{
		StartOffset: kafkaGo.FirstOffset,
		Brokers:     s.brokers,
		GroupID:     group,
		Topic:       topic,
		MinBytes:    10e3, // 10KB fetch.min.bytes 服务器应该为获取请求返回的最小数据量。如果没有足够的数据可用，请求将等待那么多数据累积后再响应请求
		MaxBytes:    10e6, // 10MB
		// CommitInterval: time.Second,           // 提交间隔指示将偏移提交到代理的间隔。如果为 0，则同步处理提交。 要加 GroupID 才能自动提交
		MaxWait: 10 * time.Millisecond, // fetch.max.wait.ms 默认10s如果没有足够的数据立即满足fetch.min.bytes提供的要求，服务器在响应fetch请求之前将阻塞的最长时间
	})
	for {
		msg, err := reader.FetchMessage(ctx)
		// io.EOF means sub closed
		// io.ErrClosedPipe means committing messages on the sub,
		// kafka will refire the messages on uncommitted messages, ignore
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			break
		}
		if err != nil {
			fmt.Printf("--SubFetch--: %v \n", err)
			break
		}

		// 从消息 headers 提取 trace context,创建 child span
		msgCtx := otel.GetTextMapPropagator().Extract(context.Background(), &kafkaHeadersCarrier{headers: &msg.Headers})
		tracer := otel.Tracer("kafka-consumer")
		msgCtx, span := tracer.Start(msgCtx, fmt.Sprintf("consume %s", topic))

		key, value := string(msg.Key), string(msg.Value)
		consumeErr := consumeWithRetry(ctx, key, value, func(_ context.Context, k, v string) error {
			return handler(msgCtx, k, v)
		})
		span.End()

		// ctx 取消(进程退出)时直接退出,不提交 offset,让 kafka 重投未处理完的消息。
		if consumeErr != nil && ctx.Err() != nil {
			break
		}
		if consumeErr != nil {
			fmt.Printf("poison message skipped after %d retries, key=%s err: %v \n", maxConsumeRetry, key, consumeErr)
		}
		if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
			fmt.Printf("commit failed, error: %v \n", commitErr)
		}
	}
}

// consumeWithRetry 对单条消息按指数退避重试,ctx 取消时立即返回。
// 超过 maxConsumeRetry 仍失败则返回最后一次错误,由调用方决定是否跳过(提交 offset)。
func consumeWithRetry(ctx context.Context, key, value string, handler func(ctx context.Context, key, value string) error) error {
	var lastErr error
	backoff := 100 * time.Millisecond
	for i := 0; i < maxConsumeRetry; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := handler(ctx, key, value); err != nil {
			lastErr = err
			fmt.Printf("consume retry %d/%d, key=%s err: %v \n", i+1, maxConsumeRetry, key, err)
			// 退避等待,ctx 取消则提前退出
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}
		return nil
	}
	return lastErr
}

// getOrCreateReader 按 topic:group 复用 reader。
// 同一 topic:group 仅创建一次并缓存到 box,Close 时统一关闭,避免每次订阅都泄漏一个 reader。
func (s *Subscriber) getOrCreateReader(topic, group string, config kafkaGo.ReaderConfig) *kafkaGo.Reader {
	key := fmt.Sprintf(topicGroupKey, topic, group)
	if v, ok := s.box.Load(key); ok {
		return v.(*kafkaGo.Reader)
	}
	reader := kafkaGo.NewReader(config)
	actual, _ := s.box.LoadOrStore(key, reader)
	return actual.(*kafkaGo.Reader)
}

func (s *Subscriber) Close() {
	var wg sync.WaitGroup
	s.box.Range(func(_, value any) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = value.(*kafkaGo.Reader).Close()
		}()
		return true
	})
	// 等待所有 reader 关闭完成,避免 offset 未提交完就退出导致消息重复消费
	wg.Wait()
}
