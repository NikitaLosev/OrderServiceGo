package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"LZero/pkg/models"
)

func TestCacheConcurrency(t *testing.T) {
	c := New(5 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.StartJanitor(ctx, time.Second)

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			o := models.Order{OrderUID: fmt.Sprintf("%d", i)}
			c.Set(o.OrderUID, o)
			if _, ok := c.Get(o.OrderUID); !ok {
				t.Errorf("order %d missing", i)
			}
		}(i)
	}
	wg.Wait()
}

func TestCacheTTL(t *testing.T) {
	c := New(100 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	c.StartJanitor(ctx, 50*time.Millisecond)
	defer cancel()
	o := models.Order{OrderUID: "ttl"}
	c.Set("ttl", o)
	if _, ok := c.Get("ttl"); !ok {
		t.Fatalf("order missing immediately")
	}
	time.Sleep(200 * time.Millisecond)
	if _, ok := c.Get("ttl"); ok {
		t.Fatalf("order not expired")
	}
}
