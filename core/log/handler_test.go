package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func parseLogLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("日志不是合法 JSON: %v (out=%s)", err, buf.String())
	}
	return out
}

func testSpanContext(t *testing.T) trace.SpanContext {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
}

func TestHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf}).
		With("service", "tituskit")

	logger.Info("hello")

	out := parseLogLine(t, &buf)
	if out["service"] != "tituskit" {
		t.Errorf("预置属性丢失: %v", out)
	}
	if out["msg"] != "hello" {
		t.Errorf("msg = %v", out["msg"])
	}
}

// WithAttrs 返回的新 Handler 仍然要注入 trace/span。
func TestHandlerWithAttrsKeepsTraceInjection(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf}).
		With("service", "tituskit")

	sc := testSpanContext(t)
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	logger.InfoContext(ctx, "hello")

	out := parseLogLine(t, &buf)
	if out["trace"] != sc.TraceID().String() {
		t.Errorf("trace = %v, 期望 %v", out["trace"], sc.TraceID().String())
	}
	if out["span"] != sc.SpanID().String() {
		t.Errorf("span = %v, 期望 %v", out["span"], sc.SpanID().String())
	}
	if out["service"] != "tituskit" {
		t.Errorf("预置属性丢失: %v", out)
	}
}

func TestHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf}).
		WithGroup("http")

	logger.Info("request", "method", "GET")

	out := parseLogLine(t, &buf)
	group, ok := out["http"].(map[string]any)
	if !ok {
		t.Fatalf("分组丢失: %v", out)
	}
	if group["method"] != "GET" {
		t.Errorf("分组内属性 = %v, 期望 GET", group["method"])
	}
}

// WithGroup 之后仍然注入 trace/span。
func TestHandlerWithGroupKeepsTraceInjection(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf}).
		WithGroup("http")

	sc := testSpanContext(t)
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	logger.InfoContext(ctx, "request")

	// 分组生效时 trace 会落在分组内。
	out := parseLogLine(t, &buf)
	if group, ok := out["http"].(map[string]any); ok {
		if group["trace"] != sc.TraceID().String() {
			t.Errorf("分组内 trace = %v, 期望 %v", group["trace"], sc.TraceID().String())
		}
		return
	}
	if out["trace"] != sc.TraceID().String() {
		t.Errorf("trace = %v, 期望 %v", out["trace"], sc.TraceID().String())
	}
}

// NewHandler 包装的 Handler 必须实现 slog.Handler 的全部方法。
func TestNewHandlerImplementsInterface(t *testing.T) {
	var handler slog.Handler = NewHandler(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	if handler.WithAttrs([]slog.Attr{slog.String("k", "v")}) == nil {
		t.Fatal("WithAttrs 返回 nil")
	}
	if handler.WithGroup("g") == nil {
		t.Fatal("WithGroup 返回 nil")
	}
}

func TestHandlerEnabledDelegatesToInner(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	handler := NewHandler(inner)

	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Info 级别应被内层 handler 拒绝")
	}
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("Error 级别应被内层 handler 接受")
	}
}
