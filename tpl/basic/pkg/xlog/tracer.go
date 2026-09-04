/**
 * @Desc: OpenTelemetry TracerProvider 初始化
 */

package xlog

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

// TracerProvider 暴露 sdk 类型别名,调用方无需直接 import otel sdk。
type TracerProvider = sdktrace.TracerProvider

// TracingConfig 追踪配置
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
	// SampleRate 采样率 0.0~1.0;<=0 时使用默认 0.1(10%)。
	SampleRate float64
}

// InitTracer 初始化追踪器
func InitTracer(cfg TracingConfig) *TracerProvider {
	if cfg.ServiceName == "" {
		return nil
	}

	// 采样率:默认 10%,可通过配置覆盖。
	ratio := cfg.SampleRate
	if ratio <= 0 {
		ratio = 0.1
	}
	if ratio > 1 {
		ratio = 1
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
			// 采样率:从配置读取(默认 0.1=10%);1.0 为全量采样,适合调试。
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(ratio)),
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
