package protocol

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

type HookEvent struct {
	EventName  string          `json:"event_name"`
	ThreadID   string          `json:"thread_id,omitempty"`
	TurnID     string          `json:"turn_id,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	Summary    string          `json:"summary,omitempty"`
	Details    string          `json:"details,omitempty"`
	HookRunID  string          `json:"hook_run_id,omitempty"`
	SourcePath string          `json:"source_path,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

func (e HookEvent) Validate() error {
	if strings.TrimSpace(e.EventName) == "" {
		return fmt.Errorf("event_name is required")
	}
	return nil
}

func ToNotification(event HookEvent) (notifier.Event, bool) {
	switch event.EventName {
	case "stop":
		return stopNotification(event), true
	default:
		return notifier.Event{}, false
	}
}

func stopNotification(event HookEvent) notifier.Event {
	body := firstNonEmpty(
		strings.TrimSpace(event.Summary),
		"Codex stopped and is waiting for user action",
	)
	if details := strings.TrimSpace(event.Details); details != "" {
		body = fmt.Sprintf("%s (%s)", body, truncate(details, 120))
	}

	return notifier.Event{
		Category: notifier.CategoryActionRequired,
		Subtitle: "Codex Stopped",
		Body:     body,
		Key:      strings.Join([]string{event.EventName, event.ThreadID, event.TurnID, event.HookRunID}, "|"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}
