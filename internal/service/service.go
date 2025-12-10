package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"LZero/internal/observability"
	"LZero/internal/repository"
	"LZero/pkg/models"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	repo     repository.OrderRepository
	cache    repository.CacheRepository
	cacheTTL time.Duration
	logger   *slog.Logger
	tracer   trace.Tracer
}

func New(repo repository.OrderRepository, cache repository.CacheRepository, cacheTTL time.Duration, logger *slog.Logger, tracer trace.Tracer) *Service {
	return &Service{
		repo:     repo,
		cache:    cache,
		cacheTTL: cacheTTL,
		logger:   logger,
		tracer:   tracer,
	}
}

func (s *Service) SaveOrder(ctx context.Context, order models.Order) error {
	if err := ValidateOrder(order); err != nil {
		return err
	}
	ctx, span := s.tracer.Start(ctx, "service.SaveOrder")
	defer span.End()

	logger := s.logger
	if reqID := observability.RequestIDFromContext(ctx); reqID != "" {
		logger = logger.With("req_id", reqID)
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return fmt.Errorf("save order: %w", err)
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, order.OrderUID, order, s.cacheTTL); err != nil {
			logger.Error("cache set failed", "err", err, "uid", order.OrderUID)
		}
	}
	span.SetAttributes(attribute.String("order_uid", order.OrderUID))
	return nil
}

func (s *Service) GetOrder(ctx context.Context, uid string) (models.Order, error) {
	if uid == "" {
		return models.Order{}, ErrValidation
	}
	ctx, span := s.tracer.Start(ctx, "service.GetOrder")
	defer span.End()

	if s.cache != nil {
		if cached, ok, err := s.cache.Get(ctx, uid); err == nil && ok {
			span.SetAttributes(attribute.String("source", "cache"))
			return cached, nil
		} else if err != nil {
			s.logger.Error("cache get failed", "err", err, "uid", uid)
		}
	}

	logger := s.logger
	if reqID := observability.RequestIDFromContext(ctx); reqID != "" {
		logger = logger.With("req_id", reqID)
	}

	order, err := s.repo.GetOrder(ctx, uid)
	if err != nil {
		if err == repository.ErrNotFound {
			return models.Order{}, ErrNotFound
		}
		return models.Order{}, fmt.Errorf("get order: %w", err)
	}

	if s.cache != nil {
		if err := s.cache.Set(ctx, order.OrderUID, order, s.cacheTTL); err != nil {
			logger.Error("cache set failed", "err", err, "uid", uid)
		}
	}
	span.SetAttributes(attribute.String("order_uid", uid))
	return order, nil
}

func (s *Service) RestoreCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	ctx, span := s.tracer.Start(ctx, "service.RestoreCache")
	defer span.End()

	orders, err := s.repo.ListOrders(ctx)
	if err != nil {
		return fmt.Errorf("list orders: %w", err)
	}
	for _, o := range orders {
		if err := s.cache.Set(ctx, o.OrderUID, o, s.cacheTTL); err != nil {
			s.logger.Error("cache warmup failed", "err", err, "uid", o.OrderUID)
		}
	}
	span.SetAttributes(attribute.Int("cache_primed", len(orders)))
	return nil
}
