package db

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// recordHandler 把 slog 记录收集到内存，便于断言级别与字段。
type recordHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *recordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordHandler) WithGroup(string) slog.Handler      { return h }

func (h *recordHandler) all() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record(nil), h.records...)
}

func (h *recordHandler) single(t *testing.T) slog.Record {
	t.Helper()
	recs := h.all()
	if len(recs) != 1 {
		t.Fatalf("期望 1 条日志，实际 %d 条", len(recs))
	}
	return recs[0]
}

func attrsOf(r slog.Record) map[string]any {
	m := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})
	return m
}

// captureSlog 把默认 logger 指向内存记录器，用例结束后还原。
func captureSlog(t *testing.T) *recordHandler {
	t.Helper()
	h := &recordHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return h
}

func TestNewSlogGormLogger(t *testing.T) {
	iface := newSlogGormLogger(time.Second)
	if iface == nil {
		t.Fatal("newSlogGormLogger 返回 nil")
	}
	if iface.LogMode(gormlogger.Info) != iface {
		t.Fatal("LogMode 应返回自身，保证 GORM 调整日志级别后行为一致")
	}
}

func TestSlogGormLoggerLevels(t *testing.T) {
	h := captureSlog(t)
	l := newSlogGormLogger(time.Second)

	l.Info(context.Background(), "info %s", "msg")
	l.Warn(context.Background(), "warn %s", "msg")
	l.Error(context.Background(), "error %s", "msg")

	recs := h.all()
	if len(recs) != 3 {
		t.Fatalf("期望 3 条日志，实际 %d 条", len(recs))
	}

	want := []struct {
		level slog.Level
		msg   string
	}{
		{slog.LevelInfo, "info msg"},
		{slog.LevelWarn, "warn msg"},
		{slog.LevelError, "error msg"},
	}
	for i, w := range want {
		if recs[i].Level != w.level || recs[i].Message != w.msg {
			t.Errorf("第 %d 条 = (%v, %q), 期望 (%v, %q)",
				i, recs[i].Level, recs[i].Message, w.level, w.msg)
		}
	}
}

func TestSlogGormLoggerTrace(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		elapsed     time.Duration
		wantLevel   slog.Level
		wantMsg     string
		wantErrAttr bool
	}{
		{"记录不存在", gorm.ErrRecordNotFound, 0, slog.LevelWarn, "gorm trace", true},
		{"查询报错", errors.New("boom"), 0, slog.LevelError, "gorm trace", true},
		{"慢查询", nil, time.Second, slog.LevelWarn, "gorm slow sql", false},
		{"正常查询", nil, 0, slog.LevelInfo, "gorm sql", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := captureSlog(t)
			l := &slogGormLogger{slowThreshold: 200 * time.Millisecond}

			// 传入一个足够早的 begin 来构造慢查询耗时。
			begin := time.Now().Add(-tt.elapsed)
			l.Trace(context.Background(), begin,
				func() (string, int64) { return "SELECT 1", 3 },
				tt.err)

			rec := h.single(t)
			if rec.Message != tt.wantMsg {
				t.Errorf("日志消息 = %q, 期望 %q", rec.Message, tt.wantMsg)
			}
			if rec.Level != tt.wantLevel {
				t.Errorf("日志级别 = %v, 期望 %v", rec.Level, tt.wantLevel)
			}

			attrs := attrsOf(rec)
			for _, key := range []string{"event", "sql", "rows", "elapsed"} {
				if _, ok := attrs[key]; !ok {
					t.Errorf("缺少属性 %q: %v", key, attrs)
				}
			}
			if attrs["event"] != "gorm" {
				t.Errorf("event = %v", attrs["event"])
			}
			if attrs["sql"] != "SELECT 1" {
				t.Errorf("sql = %v", attrs["sql"])
			}
			if attrs["rows"] != int64(3) {
				t.Errorf("rows = %v, 期望 3", attrs["rows"])
			}

			_, hasErrAttr := attrs["error"]
			if hasErrAttr != tt.wantErrAttr {
				t.Errorf("error 属性存在 = %v, 期望 %v (%v)", hasErrAttr, tt.wantErrAttr, attrs)
			}
		})
	}
}
