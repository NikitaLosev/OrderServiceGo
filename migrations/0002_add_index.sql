-- +goose Up
CREATE INDEX IF NOT EXISTS idx_orders_customer_date ON orders (customer_id, date_created);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_customer_date;
