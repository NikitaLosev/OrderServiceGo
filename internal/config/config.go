package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPAddr       string        `env:"HTTP_ADDR" env-default:":8081"`
	GRPCAddr       string        `env:"GRPC_ADDR" env-default:":9090"`
	KafkaBrokers   []string      `env:"KAFKA_BROKERS" env-separator:"," env-required:"true"`
	KafkaTopic     string        `env:"KAFKA_TOPIC" env-default:"orders_topic"`
	DatabaseURL    string        `env:"DATABASE_URL" env-required:"true"`
	RedisAddr      string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
	RedisPassword  string        `env:"REDIS_PASSWORD" env-default:""`
	CacheTTL       time.Duration `env:"CACHE_TTL" env-default:"5m"`
	JaegerEndpoint string        `env:"JAEGER_ENDPOINT" env-default:"http://localhost:14268/api/traces"`
	ServiceName    string        `env:"SERVICE_NAME" env-default:"orders-service"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
