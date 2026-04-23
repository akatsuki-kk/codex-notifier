package protocol

import (
	"encoding/json"
	"testing"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

func TestToNotificationFromAppServerCommandApproval(t *testing.T) {
	// コマンド承認要求を日本語の介入要求通知へ変換できることを確認する。
	params := json.RawMessage(`{
		"itemId":"item-1",
		"threadId":"thread-1",
		"turnId":"turn-1",
		"command":"rm -rf tmp"
	}`)

	event, ok := ToNotificationFromAppServer("item/commandExecution/requestApproval", params, "101")
	if !ok {
		t.Fatal("expected notification")
	}
	if event.Category != notifier.CategoryActionRequired {
		t.Fatalf("unexpected category: %s", event.Category)
	}
	if event.Subtitle != "コマンド実行の確認待ち" {
		t.Fatalf("unexpected subtitle: %s", event.Subtitle)
	}
}

func TestToNotificationFromAppServerTurnCompleted(t *testing.T) {
	// turn/completed を完了通知へ変換できることを確認する。
	params := json.RawMessage(`{
		"turn":{"id":"turn-1","threadId":"thread-1","status":"completed"}
	}`)

	event, ok := ToNotificationFromAppServer("turn/completed", params, "")
	if !ok {
		t.Fatal("expected notification")
	}
	if event.Category != notifier.CategoryTurnCompleted {
		t.Fatalf("unexpected category: %s", event.Category)
	}
}

func TestToNotificationFromAppServerThreadStatusChanged(t *testing.T) {
	// waitingOnApproval を含む thread/status/changed を介入要求通知へ変換できることを確認する。
	params := json.RawMessage(`{
		"threadId":"thread-1",
		"status":{"type":"active","activeFlags":["waitingOnApproval"]}
	}`)

	event, ok := ToNotificationFromAppServer("thread/status/changed", params, "")
	if !ok {
		t.Fatal("expected notification")
	}
	if event.Subtitle != "ユーザー確認待ち" {
		t.Fatalf("unexpected subtitle: %s", event.Subtitle)
	}
}
