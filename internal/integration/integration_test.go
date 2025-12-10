//go:build integration

package integration

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"LZero/internal/consumer"
	"LZero/internal/db"
	"LZero/internal/observability"
	"LZero/internal/producer"
	"LZero/internal/repository/postgres"
	redisrepo "LZero/internal/repository/redis"
	"LZero/internal/service"
	"LZero/pkg/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	tcKafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"go.opentelemetry.io/otel"
)

func TestKafkaToPostgresFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	observability.InitTracer(ctx, "orders-service-test", "")
	tracer := otel.Tracer("orders-service-test")

	pgContainer, err := tcPostgres.RunContainer(ctx,
		tcPostgres.WithDatabase("orders"),
		tcPostgres.WithUsername("user"),
		tcPostgres.WithPassword("pass"),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	applyMigrations(t, dsn)

	redisContainer, err := tcRedis.RunContainer(ctx)
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx) //nolint:errcheck

	redisEndpoint, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)
	redisAddr := strings.TrimPrefix(redisEndpoint, "redis://")

	kafkaContainer, err := tcKafka.RunContainer(ctx)
	require.NoError(t, err)
	defer kafkaContainer.Terminate(ctx) //nolint:errcheck

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	topic := "orders_topic"

	conn, err := kafka.Dial("tcp", brokers[0])
	require.NoError(t, err)
	require.NoError(t, conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}))
	conn.Close()

	pool, err := db.ConnectDB(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()

	orderRepo := postgres.NewOrderRepository(pool, tracer)
	cacheRepo := redisrepo.NewOrderCache(redisClient, tracer)
	svc := service.New(orderRepo, cacheRepo, time.Minute, logger, tracer)

	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	go func() {
		save := func(ctx context.Context, o models.Order) error { return svc.SaveOrder(ctx, o) }
		_ = consumer.StartKafkaConsumer(consumeCtx, brokers, topic, "integration", save, logger, tracer)
	}()

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	order := randomOrder()
	require.NoError(t, producer.Publish(ctx, writer, order))

	require.Eventually(t, func() bool {
		_, err := svc.GetOrder(context.Background(), order.OrderUID)
		return err == nil
	}, 30*time.Second, time.Second, "order should be saved to DB and cache")
}

func applyMigrations(t *testing.T, dsn string) {
	t.Helper()
	dbSQL, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer dbSQL.Close()

	require.NoError(t, goose.Up(dbSQL, "migrations"))
}

func randomOrder() models.Order {
	now := time.Now().UTC()
	return models.Order{
		OrderUID:          uuid.NewString(),
		TrackNumber:       "TRK123",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "sig",
		CustomerID:        "customer1",
		DeliveryService:   "delivery",
		ShardKey:          "1",
		SmID:              1,
		DateCreated:       now,
		OofShard:          "1",
		Delivery: models.Delivery{
			Name:    "John",
			Phone:   "+1234567",
			Zip:     "000000",
			City:    "City",
			Address: "Street 1",
			Region:  "Region",
			Email:   "a@example.com",
		},
		Payment: models.Payment{
			Transaction:  uuid.NewString(),
			RequestID:    "req",
			Currency:     "USD",
			Provider:     "visa",
			Amount:       100,
			PaymentDT:    now.Unix(),
			Bank:         "bank",
			DeliveryCost: 10,
			GoodsTotal:   90,
			CustomFee:    0,
		},
		Items: []models.Item{{
			ChrtID:      1,
			TrackNumber: "TRK123",
			Price:       100,
			Rid:         "RID1",
			Name:        "Item",
			Sale:        0,
			Size:        "M",
			TotalPrice:  100,
			NmID:        1,
			Brand:       "Brand",
			Status:      1,
		}},
	}
}
