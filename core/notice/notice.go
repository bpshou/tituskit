package notice

import (
	"context"
	"log/slog"

	"github.com/gen2brain/beeep"
)

// SendNoticeAlert 发送桌面通知
func SendNoticeAlert(ctx context.Context, title, message string) error {
	// beeep.Alert 的第三个参数是图标路径，传空字符串使用默认图标
	// 如果你想播放声音，可以使用 beeep.Alert，它会自带一个警告音
	// 这里为了和 Mac 版本行为更接近（显示通知+声音），使用 beeep.Alert
	err := beeep.Alert(title, message, "")
	if err != nil {
		// beeep.Alert 内部可能调用 notify-send，如果失败则记录错误
		slog.ErrorContext(ctx, "send notification error",
			slog.Any("error", err),
			slog.String("title", title),
			slog.String("message", message),
		)
		return err
	}

	slog.InfoContext(ctx, "notification sent",
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}

// SendNoticeNotify 发送桌面通知 (不带声音)
func SendNoticeNotify(ctx context.Context, title, message string) error {
	// beeep.Notify 的第三个参数是图标路径，传空字符串使用默认图标
	err := beeep.Notify(title, message, "")
	if err != nil {
		slog.ErrorContext(ctx, "send notification error",
			slog.Any("error", err),
			slog.String("title", title),
			slog.String("message", message),
		)
		return err
	}

	slog.InfoContext(ctx, "notification sent",
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}
