package middleware

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// TracingConfig 追踪配置
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
}

// InitTracer 初始化追踪器
func InitTracer(cfg TracingConfig) *sdktrace.TracerProvider {
	if cfg.ServiceName == "" {
		return nil
	}

	// 创建资源信息
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
	)

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	// 如果配置了 OTLP 地址，则添加 OTLP 导出器
	if cfg.Endpoint != "" {
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithInsecure(),
		)

		ctx := context.Background()
		exp, err := otlptrace.New(ctx, client)
		if err != nil {
			return nil
		}

		opts = append(opts, sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		))
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		append(opts,
			// 设置采样策略：生产环境使用 10% 比例采样，避免全量采样带来的性能开销,始终采样:sdktrace.AlwaysSample()
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)),
		)...,
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 设置全局 TextMapPropagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context
		propagation.Baggage{},
	))

	return tp
}
