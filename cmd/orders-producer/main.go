package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"strings"

	"LZero/internal/observability"
	"LZero/internal/producer"
	"LZero/pkg/models"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

func main() {
	file := flag.String("f", "test.json", "path to JSON file")
	flag.Parse()

	data, err := os.ReadFile(*file)
	if err != nil {
		slog.Error("read file", "err", err)
		return
	}

	var o models.Order
	if err := json.Unmarshal(data, &o); err != nil {
		slog.Error("parse json", "err", err)
		return
	}

	var cfg producerConfig
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		slog.Error("config", "err", err)
		return
	}

	ctx := context.Background()
	tp, err := observability.InitTracer(ctx, cfg.ServiceName, cfg.JaegerEndpoint)
	if err != nil {
		slog.Error("tracer", "err", err)
		return
	}
	if tp != nil {
		defer tp.Shutdown(context.Background())
	}
	tracer := otel.Tracer(cfg.ServiceName)

	ctx = observability.WithRequestID(ctx, uuid.NewString())
	ctx, span := tracer.Start(ctx, "producer.publish")
	defer span.End()

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: strings.Split(cfg.KafkaBrokers, ","),
		Topic:   cfg.KafkaTopic,
	})
	defer w.Close()

	if err := producer.Publish(ctx, w, o); err != nil {
		slog.Error("publish", "err", err)
	}
}

type producerConfig struct {
	KafkaBrokers   string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	KafkaTopic     string `env:"KAFKA_TOPIC" env-default:"orders_topic"`
	ServiceName    string `env:"SERVICE_NAME" env-default:"orders-producer"`
	JaegerEndpoint string `env:"JAEGER_ENDPOINT" env-default:""`
}
