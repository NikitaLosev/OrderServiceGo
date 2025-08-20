// db: инициализация подключения к PostgreSQL
package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectDB создаёт и возвращает пул соединений к базе данных по ENV-конфигу
func ConnectDB() (*pgxpool.Pool, error) {
	// формирование строки подключения из переменных окружения
	get := func(k, def string) string {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
		return def
	}

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		get("DB_USER", "user"),
		get("DB_PASSWORD", "pass"),
		get("DB_HOST", "localhost"),
		get("DB_PORT", "5432"),
		get("DB_NAME", "orders"),
	)

	// создаём контекст с таймаутом для инициализации пула
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// создаём пул соединений
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	// проверяем доступность базы (ping)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping DB: %w", err)
	}
	return pool, nil
}
