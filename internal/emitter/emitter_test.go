package emitter

import (
	"testing"
	"time"
)

func TestBuildEventFromJSONInput(t *testing.T) {
	// hook の JSON 入力から主要フィールドを独自イベントへ詰め替えられることを確認する。
	now := time.Date(2026, 4, 17, 6, 0, 0, 0, time.UTC)
	input := []byte(`{
		"thread_id":"thread-1",
		"turn_id":"turn-1",
		"hook_run_id":"run-1",
		"reason":"approval pending"
	}`)

	event, err := BuildEvent("stop", input, now)
	if err != nil {
		t.Fatalf("BuildEvent returned error: %v", err)
	}
	if event.EventName != "stop" {
		t.Fatalf("unexpected event name: %s", event.EventName)
	}
	if event.ThreadID != "thread-1" {
		t.Fatalf("unexpected thread id: %s", event.ThreadID)
	}
	if event.Details != "approval pending" {
		t.Fatalf("unexpected details: %s", event.Details)
	}
}

func TestBuildEventFromTextInput(t *testing.T) {
	// hook の標準入力が JSON でなくても details に退避できることを確認する。
	event, err := BuildEvent("stop", []byte("plain text input"), time.Now().UTC())
	if err != nil {
		t.Fatalf("BuildEvent returned error: %v", err)
	}
	if event.Details != "plain text input" {
		t.Fatalf("unexpected details: %s", event.Details)
	}
}
