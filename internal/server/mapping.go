package server

import (
	"time"

	"orderservice/pkg/api/orderpb"
	"orderservice/pkg/models"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func modelToProto(o models.Order) *orderpb.Order {
	items := make([]*orderpb.Item, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, &orderpb.Item{
			ChrtId:      it.ChrtID,
			TrackNumber: it.TrackNumber,
			Price:       int32(it.Price),
			Rid:         it.Rid,
			Name:        it.Name,
			Sale:        int32(it.Sale),
			Size:        it.Size,
			TotalPrice:  int32(it.TotalPrice),
			NmId:        it.NmID,
			Brand:       it.Brand,
			Status:      int32(it.Status),
		})
	}
	return &orderpb.Order{
		OrderUid:          o.OrderUID,
		TrackNumber:       o.TrackNumber,
		Entry:             o.Entry,
		Delivery:          modelToProtoDelivery(o.Delivery),
		Payment:           modelToProtoPayment(o.Payment),
		Items:             items,
		Locale:            o.Locale,
		InternalSignature: o.InternalSignature,
		CustomerId:        o.CustomerID,
		DeliveryService:   o.DeliveryService,
		Shardkey:          o.ShardKey,
		SmId:              int32(o.SmID),
		DateCreated:       timestamppb.New(o.DateCreated),
		OofShard:          o.OofShard,
	}
}

func protoToModel(o *orderpb.Order) models.Order {
	items := make([]models.Item, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, models.Item{
			ChrtID:      it.ChrtId,
			TrackNumber: it.TrackNumber,
			Price:       int(it.Price),
			Rid:         it.Rid,
			Name:        it.Name,
			Sale:        int(it.Sale),
			Size:        it.Size,
			TotalPrice:  int(it.TotalPrice),
			NmID:        it.NmId,
			Brand:       it.Brand,
			Status:      int(it.Status),
		})
	}
	return models.Order{
		OrderUID:          o.OrderUid,
		TrackNumber:       o.TrackNumber,
		Entry:             o.Entry,
		Delivery:          protoToModelDelivery(o.Delivery),
		Payment:           protoToModelPayment(o.Payment),
		Items:             items,
		Locale:            o.Locale,
		InternalSignature: o.InternalSignature,
		CustomerID:        o.CustomerId,
		DeliveryService:   o.DeliveryService,
		ShardKey:          o.Shardkey,
		SmID:              int(o.SmId),
		DateCreated:       fromTimestamp(o.DateCreated),
		OofShard:          o.OofShard,
	}
}

func modelToProtoDelivery(d models.Delivery) *orderpb.Delivery {
	return &orderpb.Delivery{
		Name:    d.Name,
		Phone:   d.Phone,
		Zip:     d.Zip,
		City:    d.City,
		Address: d.Address,
		Region:  d.Region,
		Email:   d.Email,
	}
}

func modelToProtoPayment(p models.Payment) *orderpb.Payment {
	return &orderpb.Payment{
		Transaction:  p.Transaction,
		RequestId:    p.RequestID,
		Currency:     p.Currency,
		Provider:     p.Provider,
		Amount:       int32(p.Amount),
		PaymentDt:    p.PaymentDT,
		Bank:         p.Bank,
		DeliveryCost: int32(p.DeliveryCost),
		GoodsTotal:   int32(p.GoodsTotal),
		CustomFee:    int32(p.CustomFee),
	}
}

func protoToModelDelivery(d *orderpb.Delivery) models.Delivery {
	if d == nil {
		return models.Delivery{}
	}
	return models.Delivery{
		Name:    d.Name,
		Phone:   d.Phone,
		Zip:     d.Zip,
		City:    d.City,
		Address: d.Address,
		Region:  d.Region,
		Email:   d.Email,
	}
}

func protoToModelPayment(p *orderpb.Payment) models.Payment {
	if p == nil {
		return models.Payment{}
	}
	return models.Payment{
		Transaction:  p.Transaction,
		RequestID:    p.RequestId,
		Currency:     p.Currency,
		Provider:     p.Provider,
		Amount:       int(p.Amount),
		PaymentDT:    p.PaymentDt,
		Bank:         p.Bank,
		DeliveryCost: int(p.DeliveryCost),
		GoodsTotal:   int(p.GoodsTotal),
		CustomFee:    int(p.CustomFee),
	}
}

func fromTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
