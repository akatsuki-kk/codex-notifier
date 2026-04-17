package protocol

import (
	"fmt"
	"strings"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

func ManualNotification(kind, summary, details string) (notifier.Event, error) {
	normalizedKind := strings.TrimSpace(kind)
	if normalizedKind == "" {
		return notifier.Event{}, fmt.Errorf("kind is required")
	}

	body := strings.TrimSpace(summary)
	if body == "" {
		return notifier.Event{}, fmt.Errorf("summary is required")
	}

	if trimmed := strings.TrimSpace(details); trimmed != "" {
		body = fmt.Sprintf("%s (%s)", body, truncate(trimmed, 120))
	}

	return notifier.Event{
		Category: notifier.CategoryActionRequired,
		Subtitle: manualSubtitle(normalizedKind),
		Body:     body,
		Key:      "",
	}, nil
}

func manualSubtitle(kind string) string {
	switch kind {
	case "approval-pending":
		return "Approval Needed"
	case "mcp-approval-pending":
		return "MCP Approval Needed"
	case "permission-request-pending":
		return "Permission Request"
	case "skill-approval-pending":
		return "Skill Approval Needed"
	default:
		return "Codex Attention Needed"
	}
}
