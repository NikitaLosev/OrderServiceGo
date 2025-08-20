# OrderService


Микросервис для хранения и выдачи данных о заказе по его уникальному идентификатору.  
Заказы поступают из Kafka, сохраняются в PostgreSQL и кешируются в памяти для быстрого чтения.  
Сервис предоставляет HTTP‑API `GET /order/{order_uid}` и простую веб‑страницу из каталога `static`.

---

## Содержание

1. [Стек и особенности](#стек-и-особенности)  
2. [Структура репозитория](#структура-репозитория)  
3. [Архитектура и основные пакеты](#архитектура-и-основные-пакеты)  
4. [Конфигурация](#конфигурация)  
5. [Запуск локально](#запуск-локально)  
6. [API](#api)  
7. [Тестирование и CI](#тестирование-и-ci)  
8. [Полезные команды Makefile](#полезные-команды-makefile)  
9. [Третьи стороны и лицензии](#третьи-стороны-и-лицензии)

---

## Стек и особенности

- **Go 1.24**  
- **PostgreSQL + pgxpool** — пул соединений с таймаутами и ping при старте.  
- **Kafka (segmentio/kafka-go)** — чтение заказов консьюмером с коммитом оффсета только после успешного сохранения.  
- **slog** — структурированное логирование.  
- **validator/v10** — валидация DTO (заглушка лежит в `third_party` для офлайн‑сборки).  
- **Потокобезопасный кеш** — `sync.RWMutex` + TTL и джанитор для удаления просроченных записей.  
- **Graceful shutdown** — `context` + `os/signal`; корректное закрытие HTTP‑сервера, Kafka‑консьюмера и БД.  
- **Единая обработка ошибок** — маппинг доменных ошибок на HTTP‑коды, обёртывание `fmt.Errorf("…: %w")`.  

---

## Структура репозитория

```
OrderServiceGo/
├── cmd/                    # entry‑points
│   ├── orders-service      # основной сервис
│   └── orders-producer     # вспомогательный продюсер для Kafka
├── internal/               # приватные пакеты
│   ├── cache               # кеш заказов (TTL, janitor)
│   ├── config              # загрузка env‑конфига
│   ├── consumer            # Kafka‑консьюмер
│   ├── db                  # подключение к PostgreSQL
│   ├── producer            # отправка сообщений в Kafka
│   ├── server              # HTTP‑сервер и middleware
│   └── service             # бизнес‑логика, валидация, работа с БД
├── pkg/
│   └── models              # структуры данных (Order, Delivery, Payment, Item)
├── schema/                 # SQL‑схема БД
├── static/                 # простая HTML/JS‑страничка для проверки API
├── third_party/            # локальные заглушки gofakeit и validator
├── Makefile                # вспомогательные команды
├── docker-compose.yml      # Kafka + ZooKeeper
├── env.example             # пример .env
└── test.json               # пример заказа для тестов/продюсера

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

## Архитектура и основные пакеты

### `cmd/orders-service`
Точка входа сервиса:
- загрузка конфигурации;
- подключение к БД и восстановление кеша;
- запуск Kafka‑консьюмера и HTTP‑сервера в отдельных горутинах;
- ожидание сигнала `os.Interrupt` и корректное завершение всех компонентов.

### `internal/config`
Простая загрузка переменных окружения с дефолтами (`HTTP_ADDR`, `KAFKA_BROКERS`, `KAFKA_TOPIC` и параметры БД).

### `internal/db`
Создание пула соединений `pgxpool` с контекстом и проверкой доступности (`Ping`).

### `internal/service`
- `SaveOrder` — транзакционно сохраняет заказ в PostgreSQL и кладёт его в кеш.  
- `GetOrderFromDB` — выборка заказа при cache‑miss.  
- `RestoreCache` — прогрев кеша из БД при старте.  
- `ValidateOrder` — проверка структур по тегам `validate:"required"`.  
- `errors.go` — типовые ошибки `ErrNotFound`, `ErrValidation`.  

### `internal/cache`
Потокобезопасный in‑memory кеш заказов:
- `Get/Set` c TTL;
- ленивое удаление просроченных записей и периодический janitor (`StartJanitor`);
- покрыт unit‑тестами на конкурентность и истечение TTL.

### `internal/server`
HTTP‑сервер:
- маршруты:  
  - `/` — раздача статики;  
  - `GET /order/{uid}` — выдача заказа.  
- middleware логирования (`slog`) и единый обработчик ошибок.
- `StartHTTPServer` поддерживает graceful shutdown по отмене контекста.

### `internal/consumer`
Kafka‑консьюмер:
- читает сообщения, валидирует и сохраняет заказы;
- коммит оффсета после успешного `save`;
- завершает работу по отмене контекста.

### `internal/producer`
Утилита отправки заказов в Kafka; в тестах проверяется генерация уникальных заказов (`gofakeit.UUID`).

### `pkg/models`
Структуры данных с json/validate‑тегами:
`Order`, `Delivery`, `Payment`, `Item`.

### `third_party`
Минимальные офлайн‑реализации `github.com/brianvoe/gofakeit/v7` (генерация UUID) и  
`github.com/go-playground/validator/v10` (теги `required`). В реальном окружении рекомендуется заменить на полноценные пакеты.

---

## Конфигурация

Переменные окружения (см. `env.example`):

| Переменная        | Описание                         | Значение по умолчанию |
|-------------------|----------------------------------|-----------------------|
| `DB_USER`         | пользователь PostgreSQL          | `user`                |
| `DB_PASSWORD`     | пароль                           | `pass`                |
| `DB_HOST`         | хост БД                          | `localhost`           |
| `DB_PORT`         | порт БД                          | `5432`                |
| `DB_NAME`         | имя БД                           | `orders`              |
| `HTTP_ADDR`       | адрес HTTP‑сервера               | `:8081`               |
| `KAFKA_BROKERS`   | список брокеров Kafka (через ,)  | `localhost:9092`      |
| `KAFKA_TOPIC`     | топик Kafka                      | `orders_topic`        |

---

## Запуск локально

1. **Поднять Kafka и ZooKeeper**

   ```bash
   docker compose up -d kafka
   ```

2. **Подготовить БД**

   ```bash
   psql postgres <<'SQL'
   CREATE DATABASE orders;
   CREATE USER user PASSWORD 'pass';
   GRANT ALL PRIVILEGES ON DATABASE orders TO user;
   SQL
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

   psql -U user -d orders -f schema/schema.sql
   ```

3. **Настроить переменные**

   ```bash
   cp env.example .env
   export $(grep -v '^#' .env | xargs)
   ```

4. **Запустить сервис**

   ```bash
   make run
   ```

5. **Отправить пример заказа в Kafka**

   ```bash
   go run ./cmd/orders-producer -f test.json
   ```

6. **Получить заказ по HTTP**

   ```bash
   curl http://localhost:8081/order/<order_uid> | jq
   ```

Вместо `<order_uid>` подставьте ID, который отправили продюсером.

---

## API

### `GET /order/{order_uid}`

Возвращает JSON заказа:
```bash
curl http://localhost:8081/order/b563feb7b2b84b6test | jq
```

Ответ:

```json
{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "...": "..."
}
```

**Коды ответа**

| Код | Причина                     |
|-----|-----------------------------|
| 200 | заказ найден                |
| 400 | невалидный запрос / ID      |
| 404 | заказ не найден             |
| 500 | внутренняя ошибка сервера   |

---

## Тестирование и CI

- Unit‑тесты покрывают кеш, валидацию, HTTP‑обработчики, Kafka‑консьюмера/продюсера и graceful shutdown.
- `go test ./...` — обычные тесты.  
- `go test -race ./...` — проверка на гонки.  
- `go vet` и `staticcheck` — базовый статический анализ.  
- GitHub Actions (`.github/workflows/ci.yml`) прогоняет `go vet` и `go test` на каждом push.

---

## Полезные команды Makefile

```bash
make run       # запуск сервиса
make lint      # go vet + staticcheck
make test      # go test и go test -race
make kafka-up  # поднять Kafka через docker-compose
```

---

## Третьи стороны и лицензии

Проект использует сторонние библиотеки (Kafka, pgx, slog). В каталоге `third_party` размещены минимальные офлайн‑заглушки `gofakeit` и `validator` для работы без доступа к интернету. При работе в продакшене рекомендуется заменить их на официальные модули. Все остальные зависимости находятся под открытыми лицензиями, совместимыми с MIT.

---
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

