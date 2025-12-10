package redisrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"LZero/internal/repository"
	"LZero/pkg/models"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type OrderCache struct {
	client *redis.Client
	tracer trace.Tracer
}

func NewOrderCache(client *redis.Client, tracer trace.Tracer) repository.CacheRepository {
	return &OrderCache{client: client, tracer: tracer}
}

func (c *OrderCache) Get(ctx context.Context, key string) (models.Order, bool, error) {
	ctx, span := c.tracer.Start(ctx, "redis.GetOrder")
	defer span.End()

	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return models.Order{}, false, nil
		}
		return models.Order{}, false, fmt.Errorf("redis get: %w", err)
	}
	var order models.Order
	if err := json.Unmarshal(raw, &order); err != nil {
		return models.Order{}, false, fmt.Errorf("unmarshal cache: %w", err)
	}
	span.SetAttributes(attribute.String("order_uid", order.OrderUID))
	return order, true, nil
}

func (c *OrderCache) Set(ctx context.Context, key string, value models.Order, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "redis.SetOrder")
	defer span.End()

	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	if err := c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	span.SetAttributes(attribute.String("order_uid", value.OrderUID))
	return nil
}
