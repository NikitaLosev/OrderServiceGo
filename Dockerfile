FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o orders-service ./cmd/orders-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/orders-service .
COPY static ./static

EXPOSE 8081
EXPOSE 9090

ENTRYPOINT ["./orders-service"]
