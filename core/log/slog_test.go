package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// Config 不设置 IoWriter 时必须能正常工作：
// 传入 nil Writer 会在首次写日志时 panic（invalid memory address）。
func TestNewNilWriterFallsBackToStdout(t *testing.T) {
	logger := New(Config{Level: "info", Format: FormatJSON})
	if logger == nil {
		t.Fatal("New 返回了 nil logger")
	}

	logger.Info("写入默认输出，不应 panic")
}

func TestNewWritesToProvidedWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf})
	logger.Info("hello", "k", "v")

	out := buf.String()
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("未输出日志内容: %s", out)
	}
	if !strings.Contains(out, `"k":"v"`) {
		t.Errorf("未输出自定义字段: %s", out)
	}
}

func TestNewTextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatText, IoWriter: &buf})
	logger.Info("hello")

	if out := buf.String(); !strings.Contains(out, "msg=hello") {
		t.Errorf("Text 格式输出异常: %s", out)
	}
}

func TestNewRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "warn", Format: FormatJSON, IoWriter: &buf})

	logger.Info("debug info should be dropped")
	if buf.Len() != 0 {
		t.Errorf("Info 级别日志不应输出: %s", buf.String())
	}

	logger.Warn("warn should be kept")
	if !strings.Contains(buf.String(), "warn should be kept") {
		t.Errorf("Warn 级别日志应输出: %s", buf.String())
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},         // 空字符串解析失败，回退 Info
		{"nonsense", slog.LevelInfo}, // 非法值解析失败，回退 Info
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.in); got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, 期望 %v", tt.in, got, tt.want)
		}
	}
}

// Handler 应把当前 span 的 trace_id / span_id 附加到日志上。
func TestHandlerAddsTraceContext(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf})
	logger.InfoContext(ctx, "with trace")

	out := buf.String()
	if !strings.Contains(out, traceID.String()) {
		t.Errorf("日志缺少 trace_id: %s", out)
	}
	if !strings.Contains(out, spanID.String()) {
		t.Errorf("日志缺少 span_id: %s", out)
	}
}

// 没有 span 时不应写入空的 trace 字段。
func TestHandlerWithoutTraceContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: FormatJSON, IoWriter: &buf})
	logger.InfoContext(context.Background(), "no trace")

	if out := buf.String(); strings.Contains(out, `"trace"`) {
		t.Errorf("无 span 时不应输出 trace 字段: %s", out)
	}
}
