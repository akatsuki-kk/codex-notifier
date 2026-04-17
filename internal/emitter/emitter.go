package emitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/protocol"
)

func BuildEvent(eventName string, input []byte, now time.Time) (protocol.HookEvent, error) {
	event := protocol.HookEvent{
		EventName: eventName,
		Timestamp: now.Format(time.RFC3339),
		Summary:   defaultSummary(eventName),
	}

	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return event, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(trimmed, &payload); err == nil {
		event.Raw = trimmed
		event.ThreadID = readString(payload, "thread_id", "threadId")
		event.TurnID = readString(payload, "turn_id", "turnId")
		event.HookRunID = readString(payload, "hook_run_id", "hookRunId", "id")
		event.SourcePath = readString(payload, "source_path", "sourcePath")
		event.Details = firstNonEmpty(
			readString(payload, "reason"),
			readString(payload, "prompt"),
			readString(payload, "last_assistant_message", "lastAssistantMessage"),
			readString(payload, "message"),
		)
		if event.Details == "" {
			event.Details = truncate(string(trimmed), 240)
		}
		return event, nil
	}

	event.Details = truncate(string(trimmed), 240)
	return event, nil
}

func Send(ctx context.Context, serverURL string, event protocol.HookEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected notifier status: %s", resp.Status)
	}
	return nil
}

func defaultSummary(eventName string) string {
	switch eventName {
	case "stop":
		return "Codex stopped and is waiting for user action"
	case "session_start":
		return "Codex session started"
	case "user_prompt_submit":
		return "Codex received a new user prompt"
	case "pre_tool_use":
		return "Codex is about to use a tool"
	case "post_tool_use":
		return "Codex finished using a tool"
	default:
		return fmt.Sprintf("Codex hook triggered: %s", eventName)
	}
}

func readString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
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
