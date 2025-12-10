# OrderService

Сервис принимает заказы из Kafka, сохраняет их в PostgreSQL, кладёт в Redis-кеш и отдаёт по gRPC/HTTP (через grpc-gateway). Есть миграции, метрики Prometheus, трассировка в Jaeger и Swagger UI.

## Что внутри
- Go 1.24, конфиг через `cleanenv` (строгие env-теги, см. `env.example`).
- Repository pattern: `internal/repository/postgres` (SQL), `internal/repository/redis` (кеш с TTL).
- gRPC API `order.v1.OrderService/GetOrder` + grpc-gateway (`GET /order/{order_uid}`), Swagger на `/swagger/index.html`.
- Kafka consumer (segmentio/kafka-go) с пробросом TraceID/RequestID в сервис/БД/логи.
- Миграции Goose (`migrations/0001_init.sql`, `0002_add_index.sql`), команды `make migrate-up` / `migrate-status`.
- Observability: `/metrics` (RPS, latency, 5xx), OpenTelemetry → Jaeger, structured slog + request id middleware.
- Интеграционные тесты на testcontainers (Postgres + Kafka + Redis) с тэгом `integration`.

## Быстрый старт (локально)
1) Подними инфраструктуру (Postgres, Redis, Kafka, Jaeger):
```bash
make infra-up
```
2) Применить миграции (нужен `DATABASE_URL`):
```bash
DATABASE_URL=postgres://user:pass@localhost:5432/orders?sslmode=disable make migrate-up
```
3) Запуск сервиса:
```bash
make run
```
4) Отправить пример заказа в Kafka:
```bash
go run ./cmd/orders-producer -f test.json
```
5) Прочитать заказ:
```bash
curl http://localhost:8081/order/<order_uid>
# или gRPC
grpcurl -plaintext -d '{"order_uid":"<order_uid>"}' localhost:9090 order.v1.OrderService/GetOrder
```

## Конфигурация (env)
| Переменная        | По умолчанию                                   | Описание                     |
|-------------------|------------------------------------------------|------------------------------|
| `DATABASE_URL`    | `postgres://user:pass@localhost:5432/orders?sslmode=disable` | Подключение к Postgres       |
| `HTTP_ADDR`       | `:8081`                                        | HTTP (gateway + metrics)     |
| `GRPC_ADDR`       | `:9090`                                        | gRPC сервер                  |
| `KAFKA_BROKERS`   | `localhost:9092`                               | Брокеры Kafka (через запятую)|
| `KAFKA_TOPIC`     | `orders_topic`                                 | Топик заказов                |
| `REDIS_ADDR`      | `localhost:6379`                               | Redis для кеша               |
| `REDIS_PASSWORD`  | `""`                                           | Пароль Redis                 |
| `CACHE_TTL`       | `5m`                                           | TTL кеша                     |
| `JAEGER_ENDPOINT` | `http://localhost:14268/api/traces`            | Экспорт трейсов              |
| `SERVICE_NAME`    | `orders-service`                               | Имя сервиса в трейсе/логах   |

## Структура
```
cmd/orders-service        # entrypoint (конфиг, init tracer/db/redis, gRPC+HTTP)
cmd/orders-producer       # утилита отправки заказа в Kafka
internal/config           # cleanenv конфиг
internal/consumer         # Kafka consumer (trace/req-id propagation)
internal/db               # pgxpool init
internal/observability    # tracing init, request id helpers
internal/repository       # OrderRepository (postgres) + CacheRepository (redis)
internal/server           # gRPC, grpc-gateway HTTP, middleware, metrics, swagger docs
internal/service          # бизнес-логика/валидация
pkg/api/orderpb           # сгенерённые *.pb.go
migrations                # Goose миграции
proto                     # order.proto
static                    # простая страница для ручной проверки
Dockerfile                # multistage build
```

## Observability
- Метрики: `/metrics` (Prometheus) — RPS, latency (histogram), 5xx counter.
- Трейсы: OpenTelemetry → Jaeger; TraceID и RequestID прокидываются из Kafka/HTTP в логи и запросы к БД.
- Swagger: `/swagger/index.html` (сгенерировано `make swagger`).

## Тесты
- Юнит/быстрые:
```bash
make test
```
- Интеграционные с Docker (testcontainers, тэг `integration`):
```bash
make test-integration
```

## Полезные команды Makefile
- `make infra-up` — поднять Postgres/Redis/Kafka/Jaeger из docker-compose.
- `make migrate-up` / `make migrate-status` — накат/статус миграций Goose.
- `make proto` — сгенерировать gRPC/gateway код из `proto/order.proto`.
- `make swagger` — обновить Swagger-доки в `internal/server/docs`.
- `make docker-build` — собрать образ (multistage).
