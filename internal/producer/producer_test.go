package producer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"LZero/pkg/models"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/segmentio/kafka-go"
)

type fakeWriter struct{ msgs []kafka.Message }

func (f *fakeWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	f.msgs = append(f.msgs, msgs...)
	return nil
}

func TestPublishUniqueOrders(t *testing.T) {
	w := &fakeWriter{}
	for i := 0; i < 3; i++ {
		o := fakeOrder()
		if err := Publish(context.Background(), w, o); err != nil {
			t.Fatal(err)
		}
	}
	if len(w.msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(w.msgs))
	}
	seen := make(map[string]struct{})
	for _, m := range w.msgs {
		var o models.Order
		if err := json.Unmarshal(m.Value, &o); err != nil {
			t.Fatal(err)
		}
		if _, ok := seen[o.OrderUID]; ok {
			t.Fatalf("duplicate uid %s", o.OrderUID)
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
