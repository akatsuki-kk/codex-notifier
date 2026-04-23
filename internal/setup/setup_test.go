package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreatesGlobalFiles(t *testing.T) {
	// 空の codex home に対して hook 用の初期設定ファイル一式を生成できることを確認する。
	dir := t.TempDir()
	results, err := Run(Options{
		CodexHome:      dir,
		BinaryPath:     "/tmp/codex-notifier",
		EnableAgents:   true,
		EnableStopHook: true,
		Backup:         true,
		ServerURL:      "http://127.0.0.1:8787/events",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	assertContains(t, filepath.Join(dir, "config.toml"), "codex_hooks = true")
	assertContains(t, filepath.Join(dir, "hooks.json"), "\"Stop\"")
}

func TestRunMergesExistingConfig(t *testing.T) {
	// 既存 config.toml の内容を残したまま codex_hooks を有効化できることを確認する。
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n[features]\nfoo = true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Run(Options{
		CodexHome:      dir,
		BinaryPath:     "/tmp/codex-notifier",
		EnableAgents:   false,
		EnableStopHook: true,
		Backup:         true,
		ServerURL:      "http://127.0.0.1:8787/events",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "model = \"gpt-5.4\"") {
		t.Fatalf("expected existing model setting to remain: %s", text)
	}
	if !strings.Contains(text, "foo = true") {
		t.Fatalf("expected existing features setting to remain: %s", text)
	}
	if !strings.Contains(text, "codex_hooks = true") {
		t.Fatalf("expected codex_hooks to be enabled: %s", text)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	// 同じ初期設定を 2 回実行しても 2 回目は変更なしになることを確認する。
	dir := t.TempDir()
	opts := Options{
		CodexHome:      dir,
		BinaryPath:     "/tmp/codex-notifier",
		EnableAgents:   false,
		EnableStopHook: true,
		Backup:         true,
		ServerURL:      "http://127.0.0.1:8787/events",
	}

	if _, err := Run(opts); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	results, err := Run(opts)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	for _, result := range results {
		if result.Status != "unchanged" {
			t.Fatalf("expected unchanged status, got %s for %s", result.Status, result.Path)
		}
	}
}

func assertContains(t *testing.T, path, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), expected) {
		t.Fatalf("expected %s to contain %q, got %s", path, expected, string(data))
	}
}
