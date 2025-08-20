package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"LZero/internal/service"
	"LZero/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type Reader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type readerFactory func(brokers []string, topic, groupID string) Reader

var newReader readerFactory = func(brokers []string, topic, groupID string) Reader {
	return kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: topic, GroupID: groupID})
}

func StartKafkaConsumer(ctx context.Context, brokers []string, topic, groupID string, pool *pgxpool.Pool, logger *slog.Logger) error {
	r := newReader(brokers, topic, groupID)
	defer r.Close()
	return consume(ctx, r, func(o models.Order) error { return service.SaveOrder(pool, o) }, logger)
}

func consume(ctx context.Context, r Reader, save func(models.Order) error, logger *slog.Logger) error {
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			logger.Error("read", "err", err)
			continue
		}
		var o models.Order
		if err := json.Unmarshal(msg.Value, &o); err != nil {
			logger.Error("json", "err", err)
			continue
		}
		if err := service.ValidateOrder(o); err != nil {
			logger.Error("validate", "err", err)
			continue
		}
		if err := save(o); err != nil {
			logger.Error("save", "err", err)
			continue
		}
		if err := r.CommitMessages(ctx, msg); err != nil {
			logger.Error("commit", "err", err)
		}
		logger.Info("order saved", "uid", o.OrderUID)
	}
}
