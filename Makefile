.PHONY: test lint run kafka-up

run:
	go run ./cmd/orders-service

lint:
	go vet ./...
	staticcheck ./... || true

test:
	go test ./...
	go test -race ./...

kafka-up:
	docker-compose up -d kafka
