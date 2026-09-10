package log

import (
	"io"
	"log/slog"
)

type Config struct {
	Level    string    `json:"level"`
	Format   string    `json:"format"`
	IoWriter io.Writer `json:"io_writer"`
}

// 创建 slog.Logger
func New(config Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     ParseLevel(config.Level),
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = NewHandler(slog.NewJSONHandler(config.IoWriter, opts))
	} else {
		handler = NewHandler(slog.NewTextHandler(config.IoWriter, opts))
	}
	return slog.New(handler)
}

// ParseLevel 解析日志级别字符串，如果解析失败返回默认的 Info 级别
func ParseLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		// 解析失败，返回默认的 Info 级别
		return slog.LevelInfo
	}
	return l
}
