package log

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// 保存并恢复全局 TracerProvider / Propagator，避免污染其它用例。
func preserveOTelGlobals(t *testing.T) {
	t.Helper()
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
}

func TestInitTracerReturnsShutdown(t *testing.T) {
	preserveOTelGlobals(t)

	shutdown := InitTracer("tituskit-test")
	if shutdown == nil {
		t.Fatal("InitTracer 应返回清理函数")
	}

	// 清理函数可重复调用，不应 panic。
	shutdown()
	shutdown()
}

func TestInitTracerSetsGlobals(t *testing.T) {
	preserveOTelGlobals(t)

	InitTracer("tituskit-test")

	// W3C Trace Context + Baggage 组合传播器。
	fields := otel.GetTextMapPropagator().Fields()
	want := map[string]bool{"traceparent": false, "baggage": false}
	for _, f := range fields {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("传播器缺少 %s 字段 (fields=%v)", name, fields)
		}
	}
}

// 初始化后可以创建有效 span，日志 Handler 才能拿到 trace_id。
func TestInitTracerCreatesValidSpans(t *testing.T) {
	preserveOTelGlobals(t)

	shutdown := InitTracer("tituskit-test")
	defer shutdown()

	ctx, span := otel.Tracer("tituskit-test").Start(context.Background(), "test-span")
	defer span.End()

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Fatal("span context 无效，trace_id 无法注入日志")
	}
	if !sc.HasTraceID() || !sc.HasSpanID() {
		t.Fatal("span context 缺少 trace_id 或 span_id")
	}
}

func TestInitTracerEmptyServiceName(t *testing.T) {
	preserveOTelGlobals(t)

	shutdown := InitTracer("")
	if shutdown == nil {
		t.Fatal("serviceName 为空时也应返回清理函数")
	}
	shutdown()
}

// 传播器可以正常注入/提取 traceparent。
func TestInitTracerPropagatesContext(t *testing.T) {
	preserveOTelGlobals(t)

	shutdown := InitTracer("tituskit-test")
	defer shutdown()

	ctx, span := otel.Tracer("tituskit-test").Start(context.Background(), "test-span")
	defer span.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if carrier.Get("traceparent") == "" {
		t.Fatalf("traceparent 未注入: %v", carrier)
	}

	extracted := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	if got := trace.SpanContextFromContext(extracted).TraceID(); got != span.SpanContext().TraceID() {
		t.Errorf("提取到的 trace_id = %v, 期望 %v", got, span.SpanContext().TraceID())
	}
}
