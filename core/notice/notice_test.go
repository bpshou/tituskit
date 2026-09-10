package notice

import (
	"context"
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
