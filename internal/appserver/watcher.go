package appserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/akatsuki-kk/codex-notifier/internal/dedupe"
	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
	"github.com/akatsuki-kk/codex-notifier/internal/protocol"
	"github.com/gorilla/websocket"
)

type Watcher struct {
	cfg        Config
	notifier   notifier.Notifier
	seen       *dedupe.Cache
	dialer     websocket.Dialer
	subscribed map[string]struct{}
	subMu      sync.Mutex
}

func NewWatcher(cfg Config) *Watcher {
	return &Watcher{
		cfg:        cfg,
		notifier:   notifier.NewMacOS(),
		seen:       dedupe.New(cfg.DedupeWindow),
		subscribed: map[string]struct{}{},
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	for {
		err := w.runOnce(ctx)
		if err == nil || context.Cause(ctx) != nil || ctx.Err() != nil {
			return err
		}
		log.Printf("app-server disconnected: %v", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.cfg.ReconnectInterval):
		}
	}
}

func (w *Watcher) runOnce(ctx context.Context) error {
	conn, _, err := w.dialer.DialContext(ctx, w.cfg.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("connect app-server: %w", err)
	}
	defer conn.Close()

	client := newRPCClient(conn)
	events, errs := client.Start(ctx)
	if err := client.Initialize(ctx); err != nil {
		return err
	}
	log.Printf("connected to app-server: %s", w.cfg.ServerURL)

	if err := w.subscribeLoadedThreads(ctx, client); err != nil {
		log.Printf("initial thread subscription failed: %v", err)
	}

	pollCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go w.pollLoadedThreads(pollCtx, client)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errs:
			return err
		case msg := <-events:
			if msg.Method == "thread/started" {
				w.rememberThread(msg.Params)
			}

			requestID := requestIDString(msg.ID)
			event, ok := protocol.ToNotificationFromAppServer(msg.Method, msg.Params, requestID)
			if !ok {
				continue
			}
			if !w.enabled(event.Category) || event.Key == "" && event.Body == "" {
				continue
			}
			if event.Key != "" && !w.seen.Allow(event.Key) {
				continue
			}
			if err := w.notifier.Notify(ctx, event); err != nil {
				log.Printf("notification failed: %v", err)
			}
		}
	}
}

func (w *Watcher) pollLoadedThreads(ctx context.Context, client *rpcClient) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.subscribeLoadedThreads(ctx, client); err != nil {
				log.Printf("thread subscription refresh failed: %v", err)
			}
		}
	}
}

func (w *Watcher) subscribeLoadedThreads(ctx context.Context, client *rpcClient) error {
	var result struct {
		Data []string `json:"data"`
	}
	if err := client.Request(ctx, "thread/loaded/list", map[string]any{}, &result); err != nil {
		return fmt.Errorf("thread/loaded/list: %w", err)
	}

	w.subMu.Lock()
	defer w.subMu.Unlock()

	for _, threadID := range result.Data {
		if _, ok := w.subscribed[threadID]; ok {
			continue
		}
		var resumeResult struct {
			Thread struct {
				ID string `json:"id"`
			} `json:"thread"`
		}
		if err := client.Request(ctx, "thread/resume", map[string]any{"threadId": threadID}, &resumeResult); err != nil {
			log.Printf("thread/resume failed for %s: %v", threadID, err)
			continue
		}
		w.subscribed[threadID] = struct{}{}
	}
	return nil
}

func (w *Watcher) rememberThread(params json.RawMessage) {
	var payload struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(params, &payload); err != nil || strings.TrimSpace(payload.Thread.ID) == "" {
		return
	}
	w.subMu.Lock()
	w.subscribed[payload.Thread.ID] = struct{}{}
	w.subMu.Unlock()
}

func (w *Watcher) enabled(category notifier.Category) bool {
	switch category {
	case notifier.CategoryActionRequired:
		return w.cfg.NotifyAction
	case notifier.CategoryTurnCompleted:
		return w.cfg.NotifyTurnDone
	default:
		return false
	}
}

type rpcClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	pending sync.Map
	nextID  atomic.Int64
}

func newRPCClient(conn *websocket.Conn) *rpcClient {
	return &rpcClient{conn: conn}
}

func (c *rpcClient) Initialize(ctx context.Context) error {
	var result json.RawMessage
	if err := c.Request(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "codex_notifier",
			"title":   "Codex Notifier",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"experimentalApi": true,
		},
	}, &result); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	return c.Notify(ctx, "initialized", map[string]any{})
}

func (c *rpcClient) Request(ctx context.Context, method string, params any, out any) error {
	id := c.nextID.Add(1)
	ch := make(chan protocol.AppServerEnvelope, 1)
	c.pending.Store(id, ch)
	defer c.pending.Delete(id)

	message := map[string]any{
		"method": method,
		"id":     id,
		"params": params,
	}
	if err := c.writeJSON(ctx, message); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case response := <-ch:
		if response.Error != nil {
			return fmt.Errorf("%s: %s", method, response.Error.Message)
		}
		if out != nil && len(response.Result) > 0 {
			if err := json.Unmarshal(response.Result, out); err != nil {
				return fmt.Errorf("decode %s result: %w", method, err)
			}
		}
		return nil
	}
}

func (c *rpcClient) Notify(ctx context.Context, method string, params any) error {
	return c.writeJSON(ctx, map[string]any{
		"method": method,
		"params": params,
	})
}

func (c *rpcClient) HandleResponse(msg protocol.AppServerEnvelope) bool {
	if msg.Method != "" || len(msg.ID) == 0 {
		return false
	}
	id, err := strconv.ParseInt(strings.Trim(string(msg.ID), `"`), 10, 64)
	if err != nil {
		return false
	}
	value, ok := c.pending.Load(id)
	if !ok {
		return false
	}
	ch := value.(chan protocol.AppServerEnvelope)
	ch <- msg
	return true
}

func (c *rpcClient) Start(ctx context.Context) (<-chan protocol.AppServerEnvelope, <-chan error) {
	events := make(chan protocol.AppServerEnvelope, 16)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		for {
			_ = c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				select {
				case errs <- err:
				default:
				}
				return
			}

			var msg protocol.AppServerEnvelope
			if err := json.Unmarshal(data, &msg); err != nil {
				select {
				case errs <- fmt.Errorf("decode app-server frame: %w", err):
				default:
				}
				return
			}

			if c.HandleResponse(msg) {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case events <- msg:
			}
		}
	}()

	return events, errs
}

func (c *rpcClient) writeJSON(ctx context.Context, payload any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(deadline)
	} else {
		_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}
	return c.conn.WriteJSON(payload)
}

func requestIDString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var number int64
	if err := json.Unmarshal(raw, &number); err == nil {
		return strconv.FormatInt(number, 10)
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	return strings.TrimSpace(string(raw))
}
