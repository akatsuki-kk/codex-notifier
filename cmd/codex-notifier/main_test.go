package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

func TestNotifyUsesDefaultKind(t *testing.T) {
	// kind を省略したときに approval-pending として通知されることを確認する。
	fake := &fakeNotifier{}

	if err := notify([]string{"--summary", "About to request approval"}, fake); err != nil {
		t.Fatalf("notify returned error: %v", err)
	}
	if len(fake.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fake.events))
	}
	if fake.events[0].Subtitle != "Approval Needed" {
		t.Fatalf("unexpected subtitle: %s", fake.events[0].Subtitle)
	}
}

func TestNotifyRejectsEmptySummary(t *testing.T) {
	// summary が空のときは通知を拒否することを確認する。
	fake := &fakeNotifier{}

	if err := notify([]string{"--kind", "approval-pending"}, fake); err == nil {
		t.Fatal("expected error")
	}
	if len(fake.events) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(fake.events))
	}
}

func TestInitSetupCreatesConfigFiles(t *testing.T) {
	// init が codex home 配下に初期設定ファイルを生成できることを確認する。
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex-notifier")
	if err := os.WriteFile(bin, []byte(""), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	if err := initSetup([]string{
		"--codex-home", dir,
		"--binary-path", bin,
		"--server-url", "http://127.0.0.1:8787/events",
	}); err != nil {
		t.Fatalf("initSetup returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err != nil {
		t.Fatalf("expected config.toml to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "hooks.json")); err != nil {
		t.Fatalf("expected hooks.json to exist: %v", err)
	}
}

type fakeNotifier struct {
	events []notifier.Event
}

func (f *fakeNotifier) Notify(_ context.Context, event notifier.Event) error {
	f.events = append(f.events, event)
	return nil
}
