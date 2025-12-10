.PHONY: test lint run kafka-up infra-up migrate-up migrate-status proto swagger tools docker-build

BIN := $(CURDIR)/bin
export PATH := $(BIN):$(PATH)

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

test-integration:
	go test -tags=integration ./internal/integration -count=1

infra-up:
	docker-compose up -d postgres redis kafka jaeger

migrate-up: $(BIN)/goose
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required"; exit 1)
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" $(BIN)/goose -dir migrations up

migrate-status: $(BIN)/goose
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required"; exit 1)
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" $(BIN)/goose -dir migrations status

proto: $(BIN)/protoc-gen-go $(BIN)/protoc-gen-go-grpc $(BIN)/protoc-gen-grpc-gateway
	protoc --proto_path=proto --proto_path=third_party/proto \
		--go_out=. --go_opt=module=LZero \
		--go-grpc_out=. --go-grpc_opt=module=LZero \
		--grpc-gateway_out=. --grpc-gateway_opt=module=LZero \
		proto/order.proto

swagger: $(BIN)/swag
	$(BIN)/swag fmt
	$(BIN)/swag init -g cmd/orders-service/main.go -o internal/server/docs

docker-build:
	docker build -t orders-service:latest .

tools: $(BIN)/protoc-gen-go $(BIN)/protoc-gen-go-grpc $(BIN)/protoc-gen-grpc-gateway $(BIN)/goose $(BIN)/swag

$(BIN)/protoc-gen-go:
	GOBIN=$(BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

$(BIN)/protoc-gen-go-grpc:
	GOBIN=$(BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

$(BIN)/protoc-gen-grpc-gateway:
	GOBIN=$(BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

$(BIN)/goose:
	GOBIN=$(BIN) go install github.com/pressly/goose/v3/cmd/goose@latest

$(BIN)/swag:
	GOBIN=$(BIN) go install github.com/swaggo/swag/cmd/swag@latest
