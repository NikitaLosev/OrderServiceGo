-- schema: определение таблиц для хранения заказов в БД orders_service_db
\c orders_service_db

-- orders: основная таблица заказов
CREATE TABLE IF NOT EXISTS orders (
    order_uid TEXT PRIMARY KEY,
    track_number TEXT NOT NULL,
    entry TEXT NOT NULL,
    locale TEXT,
    internal_signature TEXT,
    customer_id TEXT NOT NULL,
    delivery_service TEXT,
    shardkey TEXT,
    sm_id INT,
    date_created TIMESTAMPTZ,
    oof_shard TEXT
);

-- deliveries: информация о доставке заказа
CREATE TABLE IF NOT EXISTS deliveries (
    order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid),
    name TEXT,
    phone TEXT,
    zip TEXT,
    city TEXT,
    address TEXT,
    region TEXT,
    email TEXT
);

-- payments: информация об оплате заказа
CREATE TABLE IF NOT EXISTS payments (
    order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid),
    transaction_id TEXT,
    request_id TEXT,
    currency TEXT,
    provider TEXT,
    amount INT,
    payment_dt BIGINT,
    bank TEXT,
    delivery_cost INT,
    goods_total INT,
    custom_fee INT
);

-- items: позиции (товары) в заказе
CREATE TABLE IF NOT EXISTS items(
    id SERIAL PRIMARY KEY,
    order_uid TEXT REFERENCES orders(order_uid),
    chrt_id BIGINT,
    track_number TEXT,
    price INT,
    rid TEXT,
    name TEXT,
    sale INT,
    size TEXT,
    total_price INT,
    nm_id BIGINT,
    brand TEXT,
    status INT
);