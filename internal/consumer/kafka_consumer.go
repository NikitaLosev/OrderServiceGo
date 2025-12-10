package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"orderservice/internal/observability"
	"orderservice/pkg/models"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
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

func StartKafkaConsumer(ctx context.Context, brokers []string, topic, groupID string, save func(ctx context.Context, o models.Order) error, logger *slog.Logger, tracer trace.Tracer) error {
	r := newReader(brokers, topic, groupID)
	defer r.Close()
	return consume(ctx, r, save, logger, tracer)
}

func consume(ctx context.Context, r Reader, save func(context.Context, models.Order) error, logger *slog.Logger, tracer trace.Tracer) error {
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			logger.Error("read", "err", err)
			continue
		}

		carrier := kafkaHeaderCarrier{headers: &msg.Headers}
		msgCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
		if reqID := carrier.Get("x-request-id"); reqID != "" {
			msgCtx = observability.WithRequestID(msgCtx, reqID)
		}
		msgCtx, span := tracer.Start(msgCtx, "consumer.consume")
		var o models.Order
		if err := json.Unmarshal(msg.Value, &o); err != nil {
			logger.Error("json", "err", err)
			span.RecordError(err)
			span.End()
			continue
		}
		if err := save(msgCtx, o); err != nil {
			logger.Error("save", "err", err)
			span.RecordError(err)
			span.End()
			continue
		}
		if err := r.CommitMessages(msgCtx, msg); err != nil {
			logger.Error("commit", "err", err)
			span.RecordError(err)
		}
		span.End()
		l := logger
		if reqID := observability.RequestIDFromContext(msgCtx); reqID != "" {
			l = l.With("req_id", reqID)
		}
		l.Info("order saved", "uid", o.OrderUID, "trace_id", trace.SpanContextFromContext(msgCtx).TraceID().String())
	}
}

type kafkaHeaderCarrier struct {
	headers *[]kafka.Header
}

func (c kafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

func (c kafkaHeaderCarrier) Set(key string, value string) {
	*c.headers = append(*c.headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (c kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		keys = append(keys, h.Key)
	}
	return keys
}
