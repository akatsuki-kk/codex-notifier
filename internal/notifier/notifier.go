package notifier

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Category string

const (
	CategoryActionRequired Category = "action-required"
	CategoryTurnCompleted  Category = "turn-completed"
)

type Event struct {
	Category Category
	Subtitle string
	Body     string
	Key      string
}

type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

type MacOS struct{}

func NewMacOS() Notifier {
	return MacOS{}
}

func (MacOS) Notify(ctx context.Context, event Event) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS notifications are only supported on darwin")
	}

	script := fmt.Sprintf(
		`display notification "%s" with title "%s" subtitle "%s"`,
		escape(event.Body),
		escape("Codex Notifier"),
		escape(event.Subtitle),
	)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func escape(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}
