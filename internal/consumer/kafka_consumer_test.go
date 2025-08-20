package consumer

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"LZero/pkg/models"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/segmentio/kafka-go"
)

type fakeReader struct {
	msgs []kafka.Message
	idx  int
}

func (f *fakeReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	if f.idx >= len(f.msgs) {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	m := f.msgs[f.idx]
	f.idx++
	return m, nil
}

func (f *fakeReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error { return nil }
func (f *fakeReader) Close() error                                                    { return nil }

func TestConsume(t *testing.T) {
	var msgs []kafka.Message
	var want []models.Order
	for i := 0; i < 3; i++ {
		o := fakeOrder()
		want = append(want, o)
		b, _ := json.Marshal(o)
		msgs = append(msgs, kafka.Message{Value: b})
	}
	r := &fakeReader{msgs: msgs}
	var saved []models.Order
	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	go func() {
		_ = consume(ctx, r, func(o models.Order) error {
			saved = append(saved, o)
			if len(saved) == len(want) {
				cancel()
			}
			return nil
		}, logger)
	}()
	<-ctx.Done()
	if len(saved) != len(want) {
		t.Fatalf("saved %d, want %d", len(saved), len(want))
	}
	seen := make(map[string]struct{})
	for _, o := range saved {
		if _, ok := seen[o.OrderUID]; ok {
			t.Fatalf("duplicate %s", o.OrderUID)
		}
		seen[o.OrderUID] = struct{}{}
	}
}

func fakeOrder() models.Order {
	return models.Order{
		OrderUID:        gofakeit.UUID(),
		TrackNumber:     "tn",
		Entry:           "entry",
		Locale:          "en",
		CustomerID:      "c",
		DeliveryService: "d",
		ShardKey:        "1",
		SmID:            1,
		DateCreated:     time.Now(),
		OofShard:        "1",
		Items:           []models.Item{{ChrtID: 1, TrackNumber: "tn", Price: 1, Rid: "1", Name: "n"}},
		Delivery:        models.Delivery{Name: "n", Phone: "p", Zip: "z", City: "c", Address: "a", Region: "r", Email: "e"},
		Payment:         models.Payment{Transaction: "t", Currency: "c", Provider: "p", Amount: 1},
	}
}
