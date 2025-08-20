package producer

import (
	"context"
	"encoding/json"

	"LZero/pkg/models"
	"github.com/segmentio/kafka-go"
)

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

func Publish(ctx context.Context, w Writer, o models.Order) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafka.Message{Value: data})
}
