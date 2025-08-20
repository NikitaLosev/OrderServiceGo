package cache

import (
	"context"
	"sync"
	"time"

	"LZero/pkg/models"
)

type item struct {
	val     models.Order
	expires time.Time
}

type Cache struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]item
}

func New(ttl time.Duration) *Cache {
	c := &Cache{ttl: ttl, data: make(map[string]item)}
	return c
}

func (c *Cache) Get(key string) (models.Order, bool) {
	c.mu.RLock()
	it, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(it.expires) {
		if ok {
			c.mu.Lock()
			delete(c.data, key)
			c.mu.Unlock()
		}
		return models.Order{}, false
	}
	return it.val, true
}

func (c *Cache) Set(key string, val models.Order) {
	c.mu.Lock()
	c.data[key] = item{val: val, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *Cache) cleanup() {
	now := time.Now()
	c.mu.Lock()
	for k, it := range c.data {
		if now.After(it.expires) {
			delete(c.data, k)
		}
	}
	c.mu.Unlock()
}

// StartJanitor periodically removes expired items until ctx done.
func (c *Cache) StartJanitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()
}
