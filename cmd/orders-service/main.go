package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"LZero/internal/config"
	"LZero/internal/consumer"
	"LZero/internal/db"
	"LZero/internal/observability"
	"LZero/internal/repository/postgres"
	redisrepo "LZero/internal/repository/redis"
	"LZero/internal/server"
	_ "LZero/internal/server/docs"
	"LZero/internal/service"
	"LZero/pkg/models"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

// @title			Order Service API
// @version		1.0
// @description	REST proxy to gRPC OrderService
// @BasePath		/
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "err", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	tp, err := observability.InitTracer(ctx, cfg.ServiceName, cfg.JaegerEndpoint)
	if err != nil {
		logger.Error("tracer init", "err", err)
		return
	}
	if tp != nil {
		defer tp.Shutdown(context.Background())
	}
	tracer := otel.Tracer(cfg.ServiceName)

	pool, err := db.ConnectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect", "err", err)
		return
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer redisClient.Close()

	repo := postgres.NewOrderRepository(pool, tracer)
	cache := redisrepo.NewOrderCache(redisClient, tracer)
	svc := service.New(repo, cache, cfg.CacheTTL, logger, tracer)

	if err := svc.RestoreCache(ctx); err != nil {
		logger.Error("restore cache", "err", err)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		save := func(ctx context.Context, o models.Order) error { return svc.SaveOrder(ctx, o) }
		if err := consumer.StartKafkaConsumer(ctx, cfg.KafkaBrokers, cfg.KafkaTopic, "orders_consumer", save, logger, tracer); err != nil {
			logger.Error("consumer", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.StartGRPCServer(ctx, cfg.GRPCAddr, svc, logger, tracer); err != nil {
			logger.Error("grpc", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.StartHTTPServer(ctx, cfg.HTTPAddr, cfg.GRPCAddr, logger); err != nil {
			logger.Error("http", "err", err)
		}
	}()

	<-ctx.Done()
	wg.Wait()
}
