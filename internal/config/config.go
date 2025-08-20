package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr     string
	KafkaBrokers []string
	KafkaTopic   string
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8081"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "orders_topic"),
	}
}
