package dedupe

import (
	"sync"
	"time"
)

type Cache struct {
	window time.Duration
	mu     sync.Mutex
	items  map[string]time.Time
}

func New(window time.Duration) *Cache {
	return &Cache{
		window: window,
		items:  make(map[string]time.Time),
	}
}

func (c *Cache) Allow(key string) bool {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for k, seenAt := range c.items {
		if now.Sub(seenAt) > c.window {
			delete(c.items, k)
		}
	}

	if seenAt, ok := c.items[key]; ok && now.Sub(seenAt) <= c.window {
		return false
	}

	c.items[key] = now
	return true
}
