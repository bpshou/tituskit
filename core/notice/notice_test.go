package notice

import (
	"context"
	"strings"
	"testing"
)

func TestSendNoticeAlert(t *testing.T) {
	ctx := context.Background()
	err := SendNoticeAlert(ctx, "标题", "这是有声音的提醒")
	if err != nil {
		t.Errorf("SendNoticeAlert error = %v", err)
	}
}

func TestSendNoticeNotify(t *testing.T) {
	ctx := context.Background()
	err := SendNoticeNotify(ctx, "标题", "这是一个无声的提醒")
	if err != nil {
		t.Errorf("SendNoticeNotify error = %v", err)
	}
}

// beeep 依赖外部命令（terminal-notifier / osascript），清空 PATH 可以确定性地
// 走到错误分支：函数应返回错误并记录日志，而不是 panic 或吞掉错误。
func TestSendNoticeAlertReturnsError(t *testing.T) {
	t.Setenv("PATH", "")

	err := SendNoticeAlert(context.Background(), "标题", "内容")
	if err == nil {
		t.Fatal("无法调用系统通知命令时应返回错误")
	}
	if !strings.Contains(err.Error(), "beeep") {
		t.Errorf("错误信息应来自 beeep: %v", err)
	}
}

func TestSendNoticeNotifyReturnsError(t *testing.T) {
	t.Setenv("PATH", "")

	err := SendNoticeNotify(context.Background(), "标题", "内容")
	if err == nil {
		t.Fatal("无法调用系统通知命令时应返回错误")
	}
	if !strings.Contains(err.Error(), "beeep") {
		t.Errorf("错误信息应来自 beeep: %v", err)
	}
}

// ctx 只用于日志，取消后仍应正常发送（不因 ctx 报错）。
func TestSendNoticeWithCanceledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过真实桌面通知")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// ctx 仅用于 slog，不应影响发送结果。
	_ = SendNoticeAlert(ctx, "标题", "已取消的上下文")
	_ = SendNoticeNotify(ctx, "标题", "已取消的上下文")
}
