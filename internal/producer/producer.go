package producer

import (
	"context"
	"encoding/json"

	"LZero/internal/observability"
	"LZero/pkg/models"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

func Publish(ctx context.Context, w Writer, o models.Order) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	msg := kafka.Message{Value: data}
	carrier := kafkaHeaderCarrier{headers: &msg.Headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if reqID := observability.RequestIDFromContext(ctx); reqID != "" {
		carrier.Set("x-request-id", reqID)
	}
	return w.WriteMessages(ctx, msg)
}

type kafkaHeaderCarrier struct {
	headers *[]kafka.Header
}

func (c kafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c kafkaHeaderCarrier) Set(key, value string) {
	*c.headers = append(*c.headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (c kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		keys = append(keys, h.Key)
	}
	return keys
}
