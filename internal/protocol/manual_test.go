package protocol

import (
	"testing"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

func TestManualNotificationApprovalPending(t *testing.T) {
	// approval-pending が介入要求通知へ変換されることを確認する。
	event, err := ManualNotification("approval-pending", "About to request approval", "sandbox escalation")
	if err != nil {
		t.Fatalf("ManualNotification returned error: %v", err)
	}
	if event.Category != notifier.CategoryActionRequired {
		t.Fatalf("unexpected category: %s", event.Category)
	}
	if event.Subtitle != "Approval Needed" {
		t.Fatalf("unexpected subtitle: %s", event.Subtitle)
	}
}

func TestManualNotificationRejectsEmptySummary(t *testing.T) {
	// summary が空のときはエラーになることを確認する。
	if _, err := ManualNotification("approval-pending", "", ""); err == nil {
		t.Fatal("expected error")
	}
}
