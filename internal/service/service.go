// service: сохранение заказов в БД и работа с кешем
package service

import (
	"LZero/internal/cache"
	"LZero/pkg/models"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderCache хранит последние заказы в памяти с TTL
var OrderCache = cache.New(5 * time.Minute)

// SaveOrder сохраняет заказ в БД и обновляет кеш
func SaveOrder(pool *pgxpool.Pool, order models.Order) error {
	if err := ValidateOrder(order); err != nil {
		return err
	}
	// создаём контекст с таймаутом 5 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// начинаем транзакцию
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)
	// вставка в таблицу orders
	_, err = tx.Exec(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature,
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO NOTHING
    `, order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.ShardKey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("insert orders failed: %w", err)
	}
	// вставка в таблицу deliveries
	_, err = tx.Exec(ctx, `
        INSERT INTO deliveries (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (order_uid) DO NOTHING
    `, order.OrderUID,
		order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region,
		order.Delivery.Email)
	if err != nil {
		return fmt.Errorf("insert deliveries failed: %w", err)
	}
	// вставка в таблицу payments
	_, err = tx.Exec(ctx, `
        INSERT INTO payments (
            order_uid, transaction_id, request_id, currency, provider,
            amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO NOTHING
    `, order.OrderUID,
		order.Payment.Transaction, order.Payment.RequestID,
		order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.PaymentDT,
		order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("insert payments failed: %w", err)
	}
	// вставка в таблицу items
	for _, it := range order.Items {
		_, err = tx.Exec(ctx, `
            INSERT INTO items (
                order_uid, chrt_id, track_number, price, rid,
                name, sale, size, total_price, nm_id, brand, status
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
        `, order.OrderUID,
			it.ChrtID, it.TrackNumber, it.Price,
			it.Rid, it.Name, it.Sale, it.Size,
			it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return fmt.Errorf("insert items failed: %w", err)
		}
	}
	// завершаем транзакцию (commit)
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	// обновляем кеш
	OrderCache.Set(order.OrderUID, order)
	return nil
}

// RestoreCache загружает заказы из БД в кеш
func RestoreCache(pool *pgxpool.Pool) error {
	// получаем все заказы из таблицы orders
	rows, err := pool.Query(context.Background(), `
        SELECT
            o.order_uid,
            o.track_number,
            o.entry,
            o.locale,
            o.internal_signature,
            o.customer_id,
            o.delivery_service,
            o.shardkey,
            o.sm_id,
            o.date_created,
            o.oof_shard
        FROM orders o
    `)
	if err != nil {
		return fmt.Errorf("restoreCache: select orders failed: %w", err)
	}
	defer rows.Close()

	// наполняем кеш результатами
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry,
			&o.Locale, &o.InternalSignature, &o.CustomerID,
			&o.DeliveryService, &o.ShardKey, &o.SmID,
			&o.DateCreated, &o.OofShard,
		); err != nil {
			return fmt.Errorf("restoreCache: scan order failed: %w", err)
		}

		OrderCache.Set(o.OrderUID, o)
	}

	return rows.Err()
}
