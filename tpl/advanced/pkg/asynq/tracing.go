/**
 * @Author: spruce
 * @Date: 2026-08-11
 * @Desc: asynq 任务的 OpenTelemetry trace
 *
 * asynq.Task 不提供自由 header 字段透传 trace context(ResultWriter 是写结果用,
 * 不能在 enqueue 时携带自定义元数据),因此生产端与消费端的 span 无法做严格的父子链。
 * 这里做能做的:
 *  - enqueue 端记录 producer span(在调用方的 HTTP span 下);
 *  - process 端中间件记录 consumer span。
 * 两者在 trace 后端可分别按 task name 检索;若未来 asynq 支持 task header,
 *  再补 Inject/Extract 做严格父子链。
 */

package asynq

import (
	"context"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// startEnqueueSpan 为任务入队启动 producer span。
func startEnqueueSpan(ctx context.Context, taskName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "enqueue "+taskName,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "asynq"),
			attribute.String("messaging.destination", taskName),
		),
	)
	return ctx, span
}

// TracingMiddleware 是 asynq server 端中间件:为每个任务启动 consumer span,
// 记录 task 名称/耗时/错误,包裹真正的 handler。
func TracingMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		tracer := otel.Tracer(tracerName)
		spanCtx, span := tracer.Start(ctx, "process "+task.Type(),
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("messaging.system", "asynq"),
				attribute.String("messaging.destination", task.Type()),
				attribute.Int("messaging.message_payload_size_bytes", len(task.Payload())),
			),
		)
		defer span.End()

		err := h.ProcessTask(spanCtx, task)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	})
}
