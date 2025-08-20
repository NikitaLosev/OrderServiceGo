# OrderService

OrderService — учебный сервис для чтения сообщений из Kafka,
сохранения их в PostgreSQL и выдачи данных о заказе по его
идентификатору через HTTP API.

## Возможности

* приём сообщений с информацией о заказах через Kafka;
* сохранение заказов в PostgreSQL с идемпотентностью;
* кэширование заказов в оперативной памяти;
* REST API для получения заказа по ID;
* статическая HTML‑страница для демонстрации работы.

## Требования

* Go 1.22+
* PostgreSQL
* Kafka и ZooKeeper
* `make` (опционально)

## Структура проекта

```
cmd/                # исполняемые файлы
internal/
  db/               # подключение к БД
  consumer/         # чтение сообщений из Kafka
  service/          # бизнес‑логика и кэш
  server/           # HTTP‑сервер и статика
pkg/                # общие модели
schema/             # SQL‑схема базы данных
static/             # фронтенд страница
docker-compose.yml  # запуск Kafka и ZooKeeper
run.sh              # запуск приложения
test.json           # пример сообщения для Kafka
```

## Быстрый старт

1. **Подготовьте PostgreSQL**

   ```bash
   psql postgres <<'SQL'
   CREATE DATABASE orders_service_db;
   CREATE USER orders_service_user PASSWORD 'veryhardpassword12345';
   GRANT ALL PRIVILEGES ON DATABASE orders_service_db TO orders_service_user;
   SQL

   psql -U orders_service_user -d orders_service_db -f schema/schema.sql
   ```

2. **Запустите Kafka и ZooKeeper**

   ```bash
   docker compose up -d
   ```

3. **Запустите сервис**

   ```bash
   chmod +x run.sh   # однократно
   ./run.sh
   ```

   Ожидаемый вывод:

   ```
   Connected PG
   Cache loaded: N orders
   Kafka consumer started
   HTTP server on :8081
   ```

## Пример использования

```bash
curl http://localhost:8081/order/b563feb7b2b84b6test | jq
```

Ответ:

```json
{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}
```

Видео демонстрации: [Google Drive](https://drive.google.com/file/d/1-U-Ti53Mk0OmKOQgpkMY8NvHtHTkE16J/view?usp=sharing)

## Продюсер

Для отправки тестового заказа в Kafka:

```bash
go run ./cmd/order-producer -f test.json
```

## Технические детали

* `segmentio/kafka-go` — клиент Kafka на чистом Go
* `pgx`/`pgxpool` — высокопроизводительный драйвер PostgreSQL
* in-memory кэш на основе `map[string]Order`
* подтверждение сообщений Kafka только после успешного сохранения в БД
* SQL-конструкции `ON CONFLICT` для идемпотентности

