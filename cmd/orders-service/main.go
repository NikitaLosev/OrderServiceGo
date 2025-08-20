package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"LZero/internal/config"
	"LZero/internal/consumer"
	"LZero/internal/db"
	"LZero/internal/server"
	"LZero/internal/service"
	"LZero/pkg/models"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	raw, err := os.ReadFile("test.json")
	if err == nil {
		var o models.Order
		_ = json.Unmarshal(raw, &o)
	}

	pool, err := db.ConnectDB()
	if err != nil {
		logger.Error("db connect", "err", err)
		return
	}
	defer pool.Close()

	if err := service.RestoreCache(pool); err != nil {
		logger.Error("restore cache", "err", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := consumer.StartKafkaConsumer(ctx, cfg.KafkaBrokers, cfg.KafkaTopic, "orders_consumer", pool, logger); err != nil {
			logger.Error("consumer", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.StartHTTPServer(ctx, cfg.HTTPAddr, pool, logger); err != nil {
			logger.Error("http", "err", err)
		}
	}()

	<-ctx.Done()
	wg.Wait()
}
