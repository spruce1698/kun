/**
 * @Author: spruce
 * @Date: 2026-08-11
 * @Desc: kafka 消息的 OpenTelemetry trace context 传播
 */

package kafka

import (
	"context"

	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tracerName kafka 包内 span 使用的 tracer 名称。
const tracerName = "advanced/pkg/kafka"

// headersCarrier 让 kafka message headers 实现 propagation.TextMapCarrier,
// 以便用全局 TextMapPropagator(W3C TraceContext + Baggage)注入/提取 trace context。
// kafka Header.Value 为 []byte,这里做 string <-> []byte 转换。
type headersCarrier []kafkaGo.Header

func (h *headersCarrier) Get(key string) string {
	for _, v := range *h {
		if v.Key == key {
			return string(v.Value)
		}
	}
	return ""
}

func (h *headersCarrier) Set(key, value string) {
	for i, v := range *h {
		if v.Key == key {
			(*h)[i].Value = []byte(value)
			return
		}
	}
	*h = append(*h, kafkaGo.Header{Key: key, Value: []byte(value)})
}

func (h *headersCarrier) Keys() []string {
	keys := make([]string, 0, len(*h))
	for _, v := range *h {
		keys = append(keys, v.Key)
	}
	return keys
}

// injectSpan 把 ctx 中的 trace context 注入到 kafka message headers,
// 随消息投递给消费者,实现跨进程链路串联。返回更新后的 headers。
func injectSpan(ctx context.Context, headers []kafkaGo.Header) []kafkaGo.Header {
	carrier := headersCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, &carrier)
	return []kafkaGo.Header(carrier)
}

// extractSpan 从 kafka message headers 提取 trace context 并启动一个 consumer span。
// 返回派生 ctx(含 span)与 span 本身,调用方需 defer span.End()。
func extractSpan(ctx context.Context, msg kafkaGo.Message, operation string) (context.Context, trace.Span) {
	carrier := propagation.MapCarrier{}
	for _, h := range msg.Headers {
		carrier[h.Key] = string(h.Value)
	}
	parentCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
	tracer := otel.Tracer(tracerName)
	return tracer.Start(parentCtx, operation+" "+msg.Topic,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", msg.Topic),
			attribute.Int("messaging.kafka.partition", msg.Partition),
			attribute.String("messaging.kafka.key", string(msg.Key)),
		),
	)
}

// startProducerSpan 启动一个 producer span 并把 trace context 注入返回的 headers。
func startProducerSpan(ctx context.Context, topic string) (context.Context, trace.Span, []kafkaGo.Header) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "publish "+topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
		),
	)
	headers := injectSpan(ctx, nil)
	return ctx, span, headers
}
