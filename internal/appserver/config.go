package appserver

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	ServerURL         string
	NotifyAction      bool
	NotifyTurnDone    bool
	DedupeWindow      time.Duration
	PollInterval      time.Duration
	ReconnectInterval time.Duration
}

func NewConfig(serverURL, notifyOn string, dedupeWindow time.Duration) (Config, error) {
	if strings.TrimSpace(serverURL) == "" {
		return Config{}, fmt.Errorf("server url is required")
	}
	if dedupeWindow <= 0 {
		return Config{}, fmt.Errorf("dedupe window must be positive")
	}

	cfg := Config{
		ServerURL:         serverURL,
		DedupeWindow:      dedupeWindow,
		PollInterval:      5 * time.Second,
		ReconnectInterval: 3 * time.Second,
	}

	for _, token := range strings.Split(notifyOn, ",") {
		switch strings.TrimSpace(token) {
		case "", "none":
		case "action-required":
			cfg.NotifyAction = true
		case "turn-completed":
			cfg.NotifyTurnDone = true
		default:
			return Config{}, fmt.Errorf("unsupported notify-on category: %s", token)
		}
	}
	if !cfg.NotifyAction && !cfg.NotifyTurnDone {
		return Config{}, fmt.Errorf("at least one notify-on category must be enabled")
	}
	return cfg, nil
}
