package appserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
	"github.com/gorilla/websocket"
)

func TestWatcherReceivesTurnCompleted(t *testing.T) {
	// app-server 通知を受けて完了通知を送れることを確認する。
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()

		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read initialize: %v", err)
		}
		if err := conn.WriteJSON(map[string]any{
			"id":     msg["id"],
			"result": map[string]any{},
		}); err != nil {
			t.Fatalf("write initialize response: %v", err)
		}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read initialized: %v", err)
		}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read loaded list: %v", err)
		}
		if err := conn.WriteJSON(map[string]any{
			"id":     msg["id"],
			"result": map[string]any{"data": []string{}},
		}); err != nil {
			t.Fatalf("write loaded list response: %v", err)
		}
		if err := conn.WriteJSON(map[string]any{
			"method": "turn/completed",
			"params": map[string]any{
				"turn": map[string]any{
					"id":       "turn-1",
					"threadId": "thread-1",
					"status":   "completed",
				},
			},
		}); err != nil {
			t.Fatalf("write turn/completed: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	cfg, err := NewConfig(wsURL, "turn-completed", time.Minute)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	watcher := NewWatcher(cfg)
	fake := &fakeNotifier{}
	watcher.notifier = fake

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = watcher.Run(ctx)

	if len(fake.events) == 0 {
		t.Fatal("expected notification")
	}
	if fake.events[0].Category != notifier.CategoryTurnCompleted {
		t.Fatalf("unexpected category: %s", fake.events[0].Category)
	}
}

type fakeNotifier struct {
	events []notifier.Event
}

func (f *fakeNotifier) Notify(_ context.Context, event notifier.Event) error {
	f.events = append(f.events, event)
	return nil
}

func TestRequestIDString(t *testing.T) {
	// JSON-RPC の id を通知キー向け文字列へ変換できることを確認する。
	raw, _ := json.Marshal(101)
	if got := requestIDString(raw); got != "101" {
		t.Fatalf("unexpected request id: %s", got)
	}
}
