#!/usr/bin/env bash
export DB_USER=orders_service_user
export DB_PASSWORD=veryhardpassword12345
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=orders_service_db

go run ./cmd/orders-service