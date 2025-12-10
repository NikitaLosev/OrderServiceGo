package repository

import (
	"context"
	"time"

	"orderservice/pkg/models"
)

type OrderRepository interface {
	SaveOrder(ctx context.Context, o models.Order) error
	GetOrder(ctx context.Context, uid string) (models.Order, error)
	ListOrders(ctx context.Context) ([]models.Order, error)
}

type CacheRepository interface {
	Get(ctx context.Context, key string) (models.Order, bool, error)
	Set(ctx context.Context, key string, value models.Order, ttl time.Duration) error
}
