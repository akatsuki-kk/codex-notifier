package app

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	ListenAddr   string
	NotifyAction bool
	DedupeWindow time.Duration
}

func NewConfig(listenAddr, notifyOn string, dedupeWindow time.Duration) (Config, error) {
	if strings.TrimSpace(listenAddr) == "" {
		return Config{}, fmt.Errorf("listen address is required")
	}
	if dedupeWindow <= 0 {
		return Config{}, fmt.Errorf("dedupe window must be positive")
	}

	cfg := Config{
		ListenAddr:   listenAddr,
		DedupeWindow: dedupeWindow,
	}

	for _, token := range strings.Split(notifyOn, ",") {
		switch strings.TrimSpace(token) {
		case "", "none":
		case "action-required":
			cfg.NotifyAction = true
		default:
			return Config{}, fmt.Errorf("unsupported notify-on category: %s", token)
		}
	}

	if !cfg.NotifyAction {
		return Config{}, fmt.Errorf("at least one notify-on category must be enabled")
	}

	return cfg, nil
}
