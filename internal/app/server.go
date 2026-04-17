package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/dedupe"
	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
	"github.com/akatsuki-kk/codex-notifier/internal/protocol"
)

type Server struct {
	cfg      Config
	notifier notifier.Notifier
	seen     *dedupe.Cache
}

func NewServer(cfg Config) *Server {
	return &Server{
		cfg:      cfg,
		notifier: notifier.NewMacOS(),
		seen:     dedupe.New(cfg.DedupeWindow),
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.handleEvents)

	httpServer := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("listening on http://%s", s.cfg.ListenAddr)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var event protocol.HookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := event.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notification, ok := protocol.ToNotification(event)
	if !ok {
		log.Printf("ignoring unsupported event: %s", event.EventName)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if !s.enabled(notification.Category) {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if !s.seen.Allow(notification.Key) {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	log.Printf("event received: %s", notification.Subtitle)
	if err := s.notifier.Notify(r.Context(), notification); err != nil {
		log.Printf("notification failed: %v", err)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) enabled(category notifier.Category) bool {
	switch category {
	case notifier.CategoryActionRequired:
		return s.cfg.NotifyAction
	default:
		return false
	}
}
