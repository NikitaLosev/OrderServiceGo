package service

import (
	"testing"
	"time"

	"LZero/pkg/models"
)

func TestValidateOrder(t *testing.T) {
	o := models.Order{
		OrderUID:        "1",
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
	if err := ValidateOrder(o); err != nil {
		t.Fatalf("valid order: %v", err)
	}
	bad := models.Order{}
	if err := ValidateOrder(bad); err == nil {
		t.Fatalf("expected error for invalid order")
	}
}
