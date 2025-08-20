package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"

	"LZero/internal/config"
	"LZero/internal/producer"
	"LZero/pkg/models"
	"github.com/segmentio/kafka-go"
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

	cfg := config.Load()
	w := kafka.NewWriter(kafka.WriterConfig{Brokers: cfg.KafkaBrokers, Topic: cfg.KafkaTopic})
	defer w.Close()

	if err := producer.Publish(context.Background(), w, o); err != nil {
		slog.Error("publish", "err", err)
	}
}
