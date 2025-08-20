// service: выборка заказов из БД при cache-miss
package service

import (
	"LZero/pkg/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetOrderFromDB возвращает заказ из БД, false если не найден, или ошибку.
func GetOrderFromDB(pool *pgxpool.Pool, uid string) (models.Order, bool, error) {
	var o models.Order

	ctx := context.Background()

	// выбор основных полей заказа
	err := pool.QueryRow(ctx, `
                SELECT order_uid, track_number, entry, locale, internal_signature,
                       customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
                FROM orders WHERE order_uid=$1`, uid).Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID,
		&o.DateCreated, &o.OofShard,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, false, nil
		}
		return models.Order{}, false, fmt.Errorf("orders select: %w", err)
	}

	// загрузка информации о доставке
	if err := pool.QueryRow(ctx, `
                SELECT name, phone, zip, city, address, region, email
                FROM deliveries WHERE order_uid=$1`, uid).Scan(
		&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City,
		&o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,
	); err != nil {
		return models.Order{}, false, fmt.Errorf("deliveries select: %w", err)
	}

	// загрузка информации об оплате
	if err := pool.QueryRow(ctx, `
                SELECT transaction_id, request_id, currency, provider, amount, payment_dt,
                       bank, delivery_cost, goods_total, custom_fee
                FROM payments WHERE order_uid=$1`, uid).Scan(
		&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDT, &o.Payment.Bank, &o.Payment.DeliveryCost,
		&o.Payment.GoodsTotal, &o.Payment.CustomFee,
	); err != nil {
		return models.Order{}, false, fmt.Errorf("payments select: %w", err)
	}

	// загрузка позиций заказа
	rows, err := pool.Query(ctx, `
                SELECT chrt_id, track_number, price, rid, name, sale, size,
                       total_price, nm_id, brand, status
                FROM items WHERE order_uid=$1`, uid)
	if err != nil {
		return models.Order{}, false, fmt.Errorf("items select: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			return models.Order{}, false, fmt.Errorf("items scan: %w", err)
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return models.Order{}, false, fmt.Errorf("items rows: %w", err)
	}

	return o, true, nil
}
