package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/app"
	"github.com/akatsuki-kk/codex-notifier/internal/appserver"
	"github.com/akatsuki-kk/codex-notifier/internal/emitter"
	"github.com/akatsuki-kk/codex-notifier/internal/localrun"
	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
	"github.com/akatsuki-kk/codex-notifier/internal/protocol"
	"github.com/akatsuki-kk/codex-notifier/internal/setup"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("codex-notifier: ")

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run-local":
		if err := runLocal(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "watch-app-server":
		if err := watchAppServer(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "serve":
		if err := serve(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "emit-hook":
		if err := emitHook(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "notify":
		if err := notify(os.Args[2:], notifier.NewMacOS()); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "init":
		if err := initSetup(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func runLocal(args []string) error {
	fs := flag.NewFlagSet("run-local", flag.ContinueOnError)
	worktree := fs.String("worktree", "", "target worktree; defaults to current directory")
	port := fs.Int("port", 0, "app-server port; defaults to an ephemeral localhost port")
	codexBin := fs.String("codex-bin", "codex", "codex executable path")
	notifyOn := fs.String("notify-on", "action-required,turn-completed", "comma-separated categories to notify")
	dedupeWindow := fs.Duration("dedupe-window", 30*time.Second, "dedupe window")
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return localrun.Run(ctx, localrun.Config{
		Worktree:     *worktree,
		Port:         *port,
		CodexBin:     *codexBin,
		NotifyOn:     *notifyOn,
		DedupeWindow: *dedupeWindow,
	})
}

func watchAppServer(args []string) error {
	fs := flag.NewFlagSet("watch-app-server", flag.ContinueOnError)
	serverURL := fs.String("server", "ws://127.0.0.1:4500", "Codex app-server websocket URL")
	notifyOn := fs.String("notify-on", "action-required,turn-completed", "comma-separated categories to notify")
	dedupeWindow := fs.Duration("dedupe-window", 30*time.Second, "dedupe window")
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := appserver.NewConfig(*serverURL, *notifyOn, *dedupeWindow)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	watcher := appserver.NewWatcher(cfg)
	return watcher.Run(ctx)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listenAddr := fs.String("listen", "127.0.0.1:8787", "HTTP listen address")
	notifyOn := fs.String("notify-on", "action-required", "comma-separated categories to notify")
	dedupeWindow := fs.Duration("dedupe-window", 30*time.Second, "dedupe window")
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := app.NewConfig(*listenAddr, *notifyOn, *dedupeWindow)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := app.NewServer(cfg)
	return server.Run(ctx)
}

func emitHook(args []string) error {
	fs := flag.NewFlagSet("emit-hook", flag.ContinueOnError)
	serverURL := fs.String("server", "http://127.0.0.1:8787/events", "notifier event endpoint")
	eventName := fs.String("event-name", "", "hook event name")
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *eventName == "" {
		return fmt.Errorf("--event-name is required")
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	event, err := emitter.BuildEvent(*eventName, input, time.Now().UTC())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := emitter.Send(ctx, *serverURL, event); err != nil {
		log.Printf("emit-hook warning: %v", err)
		return nil
	}
	return nil
}

func notify(args []string, sink notifier.Notifier) error {
	fs := flag.NewFlagSet("notify", flag.ContinueOnError)
	kind := fs.String("kind", "approval-pending", "manual notification kind")
	summary := fs.String("summary", "", "notification summary")
	details := fs.String("details", "", "optional notification details")
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	event, err := protocol.ManualNotification(*kind, *summary, *details)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return sink.Notify(ctx, event)
}

func initSetup(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	codexHomeDefault := os.Getenv("CODEX_HOME")
	if codexHomeDefault == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		codexHomeDefault = filepath.Join(homeDir, ".codex")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("resolve absolute executable path: %w", err)
	}

	codexHome := fs.String("codex-home", codexHomeDefault, "Codex home directory")
	binaryPath := fs.String("binary-path", exePath, "codex-notifier binary path")
	enableAgents := fs.Bool("enable-agents", true, "configure AGENTS.md and rules")
	enableStopHook := fs.Bool("enable-stop-hook", true, "configure stop hook and codex_hooks")
	backup := fs.Bool("backup", true, "write .bak files before updating existing files")
	serverURL := fs.String("server-url", "http://127.0.0.1:8787/events", "event endpoint used by stop hook")
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	binaryAbs, err := filepath.Abs(*binaryPath)
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	results, err := setup.Run(setup.Options{
		CodexHome:      *codexHome,
		BinaryPath:     binaryAbs,
		EnableAgents:   *enableAgents,
		EnableStopHook: *enableStopHook,
		Backup:         *backup,
		ServerURL:      *serverURL,
	})
	if err != nil {
		return err
	}

	for _, result := range results {
		fmt.Printf("%s: %s\n", result.Status, result.Path)
	}
	fmt.Println("next: codex-notifier serve --listen 127.0.0.1:8787")
	fmt.Println("next: restart Codex to load the updated configuration")
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  codex-notifier run-local [--worktree PATH] [--port PORT]")
	fmt.Fprintln(os.Stderr, "  codex-notifier watch-app-server --server ws://127.0.0.1:4500")
	fmt.Fprintln(os.Stderr, "  codex-notifier serve --listen 127.0.0.1:8787")
	fmt.Fprintln(os.Stderr, "  codex-notifier emit-hook --server http://127.0.0.1:8787/events --event-name stop")
	fmt.Fprintln(os.Stderr, "  codex-notifier notify --kind approval-pending --summary \"About to request approval\"")
	fmt.Fprintln(os.Stderr, "  codex-notifier init")
}
