package notifier

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
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

	cmd := macOSCommand(ctx, event)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, string(out))
	}
	return nil
}

func macOSCommand(ctx context.Context, event Event) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		"osascript",
		"-e", `on run argv`,
		"-e", `display notification (item 1 of argv) with title (item 2 of argv) subtitle (item 3 of argv)`,
		"-e", `end run`,
		event.Body,
		"Codex Notifier",
		event.Subtitle,
	)
}
