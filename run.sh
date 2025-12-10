#!/usr/bin/env bash
export DATABASE_URL=${DATABASE_URL:-postgres://user:pass@localhost:5432/orders?sslmode=disable}
export REDIS_ADDR=${REDIS_ADDR:-localhost:6379}
export KAFKA_BROKERS=${KAFKA_BROKERS:-localhost:9092}
export KAFKA_TOPIC=${KAFKA_TOPIC:-orders_topic}
export HTTP_ADDR=${HTTP_ADDR:-:8081}
export GRPC_ADDR=${GRPC_ADDR:-:9090}
export SERVICE_NAME=${SERVICE_NAME:-orders-service}

go run ./cmd/orders-service
