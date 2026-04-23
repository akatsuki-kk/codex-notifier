package localrun

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/appserver"
	"github.com/gorilla/websocket"
)

type Config struct {
	Worktree     string
	Port         int
	CodexBin     string
	NotifyOn     string
	DedupeWindow time.Duration
}

func Run(ctx context.Context, cfg Config) error {
	worktree, err := ResolveWorktree(cfg.Worktree)
	if err != nil {
		return err
	}
	port := cfg.Port
	if port == 0 {
		port, err = ChoosePort()
		if err != nil {
			return err
		}
	}
	serverURL := fmt.Sprintf("ws://127.0.0.1:%d", port)

	cmd := BuildAppServerCommand(ctx, cfg.CodexBin, worktree, serverURL)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start codex app-server: %w", err)
	}
	log.Printf("started codex app-server: pid=%d worktree=%s url=%s", cmd.Process.Pid, worktree, serverURL)

	childErr := make(chan error, 1)
	go func() {
		childErr <- cmd.Wait()
	}()

	readyCtx, cancelReady := context.WithTimeout(ctx, 15*time.Second)
	defer cancelReady()
	if err := WaitUntilReachable(readyCtx, serverURL, 200*time.Millisecond); err != nil {
		return fmt.Errorf("wait for app-server readiness: %w", err)
	}

	watcherCfg, err := appserver.NewConfig(serverURL, cfg.NotifyOn, cfg.DedupeWindow)
	if err != nil {
		return err
	}
	watcher := appserver.NewWatcher(watcherCfg)
	watcherErr := make(chan error, 1)
	go func() {
		watcherErr <- watcher.Run(ctx)
	}()

	select {
	case <-ctx.Done():
		<-childErr
		return ctx.Err()
	case err := <-childErr:
		return fmt.Errorf("codex app-server exited: %w", err)
	case err := <-watcherErr:
		return err
	}
}

func ResolveWorktree(worktree string) (string, error) {
	if worktree == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve current directory: %w", err)
		}
		worktree = cwd
	}
	abs, err := filepath.Abs(worktree)
	if err != nil {
		return "", fmt.Errorf("resolve worktree path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat worktree: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("worktree must be a directory: %s", abs)
	}
	return abs, nil
}

func ChoosePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate localhost port: %w", err)
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type")
	}
	return addr.Port, nil
}

func BuildAppServerCommand(ctx context.Context, codexBin, worktree, serverURL string) *exec.Cmd {
	if codexBin == "" {
		codexBin = "codex"
	}
	cmd := exec.CommandContext(ctx, codexBin, "app-server", "--listen", serverURL)
	cmd.Dir = worktree
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func WaitUntilReachable(ctx context.Context, serverURL string, interval time.Duration) error {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	dialer := websocket.Dialer{}

	for {
		conn, _, err := dialer.DialContext(ctx, serverURL, nil)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
