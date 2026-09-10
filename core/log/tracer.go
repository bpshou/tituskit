package log

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
)

// InitTracer 初始化 OpenTelemetry Tracer
func InitTracer(serviceName string) func() {
	// 1. 创建资源（标识服务）- 避免 Schema URL 冲突
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		slog.ErrorContext(context.Background(), "failed to create resource", slog.Any("err", err))
	}

	// 2. 创建 exporter（只生成 span 不打印数据，日志中保留 trace_id）
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(io.Discard), // 丢弃 span 详细输出
	)
	if err != nil {
		slog.ErrorContext(context.Background(), "failed to create exporter", slog.Any("err", err))
	}

	// 3. 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 全部采样（开发环境）
		// 生产环境使用：sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // 10% 采样
	)

	// 4. 设置全局 TracerProvider 和 Propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, // W3C Trace Context 标准
			propagation.Baggage{},
		),
	)

	// 5. 返回清理函数
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.ErrorContext(context.Background(), "error shutting down tracer provider", slog.Any("err", err))
		}
	}
}
