package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

func TestHandleEventsAcceptsStopEvent(t *testing.T) {
	// stop イベントを受けたときに通知が 1 回送られることを確認する。
	server := NewServer(Config{
		ListenAddr:   "127.0.0.1:8787",
		NotifyAction: true,
		DedupeWindow: time.Minute,
	})
	fake := &fakeNotifier{}
	server.notifier = fake

	body := bytes.NewBufferString(`{
		"event_name":"stop",
		"thread_id":"thread-1",
		"turn_id":"turn-1",
		"hook_run_id":"run-1",
		"summary":"Codex stop hook triggered"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", body)
	rec := httptest.NewRecorder()

	server.handleEvents(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if len(fake.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fake.events))
	}
}

func TestHandleEventsRejectsInvalidJSON(t *testing.T) {
	// 不正な JSON は 400 で拒否されることを確認する。
	server := NewServer(Config{
		ListenAddr:   "127.0.0.1:8787",
		NotifyAction: true,
		DedupeWindow: time.Minute,
	})

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	server.handleEvents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestHandleEventsDeduplicatesSameEvent(t *testing.T) {
	// 同一 stop イベントの重複通知が抑止されることを確認する。
	server := NewServer(Config{
		ListenAddr:   "127.0.0.1:8787",
		NotifyAction: true,
		DedupeWindow: time.Minute,
	})
	fake := &fakeNotifier{}
	server.notifier = fake

	payload := map[string]string{
		"event_name":  "stop",
		"thread_id":   "thread-1",
		"turn_id":     "turn-1",
		"hook_run_id": "run-1",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		server.handleEvents(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
	}

	if len(fake.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fake.events))
	}
}

type fakeNotifier struct {
	events []notifier.Event
}

func (f *fakeNotifier) Notify(_ context.Context, event notifier.Event) error {
	f.events = append(f.events, event)
	return nil
}
