package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"advanced/pkg/xconfig"
	"advanced/pkg/xlog"

	kafkaGo "github.com/segmentio/kafka-go"
)

const (
	topicGroupKey = "%s|%s:%s"
	// maxConsumeRetry 单条消息处理失败时的最大重试次数。
	// 超过后视为毒消息,跳过提交(避免无限重投),仅记录日志。
	maxConsumeRetry = 3
)

type (
	Subscriber struct {
		brokers []string
		log     *xlog.Logger // 统一日志实例,替代 fmt.Printf 便于生产检索
		box     sync.Map
	}
)

func NewSubscriber(conf *xconfig.Conf, log *xlog.Logger) *Subscriber {
	if len(conf.Kafka.Brokers) == 0 {
		panic("无可用的节点信息")
	}
	return &Subscriber{
		brokers: conf.Kafka.Brokers,
		log:     log,
		box:     sync.Map{},
	}
}

// Sub 订阅者 kafka提交 ,ctx 取消时退出消费循环。
// 同一 topic:group 仅创建一个 reader 并缓存,Close 时统一关闭。
func (s *Subscriber) Sub(ctx context.Context, topic, group string, handler func(ctx context.Context, key, value string) error) {
	reader := s.getOrCreateReader("auto", topic, group, kafkaGo.ReaderConfig{
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
			s.log.Error(ctx, "kafka sub read error", err)
			break
		}
		// 处理失败时按指数退避重试,超过 maxConsumeRetry 仍失败则跳过该消息(kafka 自动提交会推进 offset),
		// 避免handler持续报错导致消费卡死。
		s.consumeWithRetry(ctx, string(msg.Key), string(msg.Value), func(k, v string) error {
			return handler(ctx, k, v)
		})
	}
}

// SubFetch 订阅者 程序提交(速度快), ctx 取消时退出消费循环。
// 同一 topic:group 仅创建一个 reader 并缓存,Close 时统一关闭。
func (s *Subscriber) SubFetch(ctx context.Context, topic, group string, handler func(key, value string) error) {
	reader := s.getOrCreateReader("manual", topic, group, kafkaGo.ReaderConfig{
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
			s.log.Error(ctx, "kafka subfetch read error", err)
			break
		}

		// 处理失败时按指数退避重试,超过 maxConsumeRetry 仍失败则视为毒消息:
		// 仍提交 offset(跳过该消息),避免未提交导致 kafka 反复重投同一条毒消息形成死循环。
		key, value := string(msg.Key), string(msg.Value)
		consumeErr := s.consumeWithRetry(ctx, key, value, handler)
		// ctx 取消(进程退出)时直接退出,不提交 offset,让 kafka 重投未处理完的消息。
		if consumeErr != nil && ctx.Err() != nil {
			break
		}
		if consumeErr != nil {
			s.log.Warn(ctx, "poison message skipped", map[string]any{"key": key, "retries": maxConsumeRetry, "error": consumeErr.Error()})
		}
		if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
			s.log.Error(ctx, "kafka commit failed", commitErr)
		}
	}
}

// consumeWithRetry 对单条消息按指数退避重试,ctx 取消时立即返回。
// 超过 maxConsumeRetry 仍失败则返回最后一次错误,由调用方决定是否跳过(提交 offset)。
func (s *Subscriber) consumeWithRetry(ctx context.Context, key, value string, handler func(key, value string) error) error {
	var lastErr error
	backoff := 100 * time.Millisecond
	for i := 0; i < maxConsumeRetry; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := handler(key, value); err != nil {
			lastErr = err
			s.log.Warn(ctx, "consume retry", map[string]any{"attempt": i + 1, "max": maxConsumeRetry, "key": key, "error": err.Error()})
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

// getOrCreateReader 按 mode:topic:group 复用 reader。
// auto=自动提交(Sub) manual=手动提交(SubFetch) 分开缓存,
// 避免同一 topic:group 同时注册两种模式时并发读写同一个 Reader(kafka-go Reader 非并发安全)。
func (s *Subscriber) getOrCreateReader(mode, topic, group string, config kafkaGo.ReaderConfig) *kafkaGo.Reader {
	key := fmt.Sprintf(topicGroupKey, mode, topic, group)
	if v, ok := s.box.Load(key); ok {
		return v.(*kafkaGo.Reader)
	}
	reader := kafkaGo.NewReader(config)
	actual, _ := s.box.LoadOrStore(key, reader)
	return actual.(*kafkaGo.Reader)
}

func (s *Subscriber) Close() {
	s.box.Range(func(_, value any) bool {
		go func() {
			_ = value.(*kafkaGo.Reader).Close()
		}()
		return true
	})
}
