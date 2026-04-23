package localrun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveWorktreeUsesCurrentDirectoryByDefault(t *testing.T) {
	// worktree 未指定時は現在の cwd を使うことを確認する。
	dir := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(prev) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	got, err := ResolveWorktree("")
	if err != nil {
		t.Fatalf("ResolveWorktree: %v", err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks want: %v", err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks got: %v", err)
	}
	if gotResolved != want {
		t.Fatalf("unexpected worktree: %s", gotResolved)
	}
}

func TestChoosePortReturnsUsablePort(t *testing.T) {
	// 自動選択したポートが localhost で利用可能な番号になることを確認する。
	port, err := ChoosePort()
	if err != nil {
		t.Fatalf("ChoosePort: %v", err)
	}
	if port <= 0 {
		t.Fatalf("unexpected port: %d", port)
	}
}

func TestBuildAppServerCommand(t *testing.T) {
	// app-server 起動コマンドの引数と cwd を正しく組み立てることを確認する。
	dir := t.TempDir()
	cmd := BuildAppServerCommand(context.Background(), "/usr/local/bin/codex", dir, "ws://127.0.0.1:4501")

	if cmd.Dir != dir {
		t.Fatalf("unexpected dir: %s", cmd.Dir)
	}
	got := cmd.Args
	want := []string{"/usr/local/bin/codex", "app-server", "--listen", "ws://127.0.0.1:4501"}
	if len(got) != len(want) {
		t.Fatalf("unexpected arg count: %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected arg at %d: %s", i, got[i])
		}
	}
}

func TestWaitUntilReachableTimesOut(t *testing.T) {
	// 接続先が立ち上がらない場合はコンテキスト期限で終了することを確認する。
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WaitUntilReachable(ctx, "ws://127.0.0.1:65530", 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveWorktreeRejectsFile(t *testing.T) {
	// ファイルパスを worktree に指定した場合はエラーになることを確認する。
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := ResolveWorktree(path); err == nil {
		t.Fatal("expected error")
	}
}
