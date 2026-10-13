package log

import (
	"io"
	"log/slog"
	"os"
)

// 日志输出格式。
const (
	FormatJSON = "json"
	FormatText = "text"
)

type Config struct {
	Level    string    `json:"level"`
	Format   string    `json:"format"`
	IoWriter io.Writer `json:"io_writer"`
}

// New 创建 slog.Logger。
//
// IoWriter 为 nil 时默认写入 os.Stdout：slog 的 Handler 会原样保存这个
// Writer，真正写日志时才调用它的 Write，传入 nil 接口会在首次打日志时
// panic（invalid memory address），因此这里必须兜底。
//
// Format 为 FormatJSON 时输出 JSON，其余值输出 Text。
func New(config Config) *slog.Logger {
	if config.IoWriter == nil {
		config.IoWriter = os.Stdout
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     ParseLevel(config.Level),
	}

	var handler slog.Handler
	if config.Format == FormatJSON {
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
