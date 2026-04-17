package protocol

import (
	"testing"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

func TestToNotificationStop(t *testing.T) {
	// stop イベントが介入要求通知へ変換されることを確認する。
	event, ok := ToNotification(HookEvent{
		EventName: "stop",
		ThreadID:  "thread-1",
		TurnID:    "turn-1",
		HookRunID: "run-1",
		Summary:   "Codex stopped and is waiting for user action",
		Details:   "approval pending",
	})
	if !ok {
		t.Fatalf("expected notification")
	}
	if event.Category != notifier.CategoryActionRequired {
		t.Fatalf("unexpected category: %s", event.Category)
	}
	if event.Subtitle != "Codex Stopped" {
		t.Fatalf("unexpected subtitle: %s", event.Subtitle)
	}
}

func TestToNotificationUnsupportedEvent(t *testing.T) {
	// 未対応イベントは通知対象外として扱われることを確認する。
	_, ok := ToNotification(HookEvent{EventName: "session_start"})
	if ok {
		t.Fatalf("unexpected notification")
	}
}
