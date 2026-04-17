package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/app"
	"github.com/akatsuki-kk/codex-notifier/internal/emitter"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("codex-notifier: ")

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
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
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
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

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  codex-notifier serve --listen 127.0.0.1:8787")
	fmt.Fprintln(os.Stderr, "  codex-notifier emit-hook --server http://127.0.0.1:8787/events --event-name stop")
}
