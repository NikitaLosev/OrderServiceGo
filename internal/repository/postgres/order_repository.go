package postgres

import (
	"context"
	"errors"
	"fmt"

	"LZero/internal/repository"
	"LZero/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type OrderRepository struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func NewOrderRepository(pool *pgxpool.Pool, tracer trace.Tracer) *OrderRepository {
	return &OrderRepository{pool: pool, tracer: tracer}
}

func (r *OrderRepository) SaveOrder(ctx context.Context, order models.Order) error {
	ctx, span := r.tracer.Start(ctx, "postgres.SaveOrder")
	defer span.End()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature,
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO NOTHING
    `, order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.ShardKey, order.SmID, order.DateCreated, order.OofShard); err != nil {
		return fmt.Errorf("insert orders failed: %w", err)
	}

	if _, err = tx.Exec(ctx, `
        INSERT INTO deliveries (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (order_uid) DO NOTHING
    `, order.OrderUID,
		order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region,
		order.Delivery.Email); err != nil {
		return fmt.Errorf("insert deliveries failed: %w", err)
	}

	if _, err = tx.Exec(ctx, `
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
		order.Payment.GoodsTotal, order.Payment.CustomFee); err != nil {
		return fmt.Errorf("insert payments failed: %w", err)
	}

	for _, it := range order.Items {
		if _, err = tx.Exec(ctx, `
            INSERT INTO items (
                order_uid, chrt_id, track_number, price, rid,
                name, sale, size, total_price, nm_id, brand, status
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
        `, order.OrderUID,
			it.ChrtID, it.TrackNumber, it.Price,
			it.Rid, it.Name, it.Sale, it.Size,
			it.TotalPrice, it.NmID, it.Brand, it.Status); err != nil {
			return fmt.Errorf("insert items failed: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}
	span.SetAttributes(attribute.String("order_uid", order.OrderUID))
	return nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, uid string) (models.Order, error) {
	ctx, span := r.tracer.Start(ctx, "postgres.GetOrder")
	defer span.End()

	var o models.Order
	if err := r.pool.QueryRow(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature,
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid=$1`, uid).Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID,
		&o.DateCreated, &o.OofShard,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, repository.ErrNotFound
		}
		return models.Order{}, fmt.Errorf("orders select: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
        SELECT name, phone, zip, city, address, region, email
        FROM deliveries WHERE order_uid=$1`, uid).Scan(
		&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City,
		&o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,
	); err != nil {
		return models.Order{}, fmt.Errorf("deliveries select: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
        SELECT transaction_id, request_id, currency, provider, amount, payment_dt,
               bank, delivery_cost, goods_total, custom_fee
        FROM payments WHERE order_uid=$1`, uid).Scan(
		&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDT, &o.Payment.Bank, &o.Payment.DeliveryCost,
		&o.Payment.GoodsTotal, &o.Payment.CustomFee,
	); err != nil {
		return models.Order{}, fmt.Errorf("payments select: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
        SELECT chrt_id, track_number, price, rid, name, sale, size,
               total_price, nm_id, brand, status
        FROM items WHERE order_uid=$1`, uid)
	if err != nil {
		return models.Order{}, fmt.Errorf("items select: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			return models.Order{}, fmt.Errorf("items scan: %w", err)
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return models.Order{}, fmt.Errorf("items rows: %w", err)
	}
	span.SetAttributes(attribute.String("order_uid", o.OrderUID))
	return o, nil
}

func (r *OrderRepository) ListOrders(ctx context.Context) ([]models.Order, error) {
	ctx, span := r.tracer.Start(ctx, "postgres.ListOrders")
	defer span.End()

	rows, err := r.pool.Query(ctx, `
        SELECT order_uid
        FROM orders
    `)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan order uid: %w", err)
		}
		o, err := r.GetOrder(ctx, uid)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	span.SetAttributes(attribute.Int("orders_count", len(orders)))
	return orders, nil
}
