package notifier

import (
	"context"
	"testing"
)

func TestMacOSCommandPassesNotificationTextAsArguments(t *testing.T) {
	// 日本語を含む通知本文を osascript の引数として渡せることを確認する。
	cmd := macOSCommand(context.Background(), Event{
		Body:     "これから昇格権限の確認を求めます",
		Subtitle: "Codex Stopped",
	})

	got := cmd.Args
	if len(got) != 10 {
		t.Fatalf("unexpected arg count: %d", len(got))
	}
	if got[0] != "osascript" {
		t.Fatalf("unexpected command: %s", got[0])
	}
	if got[7] != "これから昇格権限の確認を求めます" {
		t.Fatalf("unexpected body arg: %s", got[7])
	}
	if got[8] != "Codex Notifier" {
		t.Fatalf("unexpected title arg: %s", got[8])
	}
	if got[9] != "Codex Stopped" {
		t.Fatalf("unexpected subtitle arg: %s", got[9])
	}
}
