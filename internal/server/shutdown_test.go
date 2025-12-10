package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"LZero/pkg/api/orderpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	grpcLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcSrv := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(grpcSrv, &stubOrderServer{})
	go grpcSrv.Serve(grpcLis)
	defer grpcSrv.GracefulStop()

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	if err := StartHTTPServer(ctx, "127.0.0.1:0", grpcLis.Addr().String(), logger); err != nil {
		t.Fatalf("server error: %v", err)
	}
}

type stubOrderServer struct {
	orderpb.UnimplementedOrderServiceServer
}

func (s *stubOrderServer) GetOrder(context.Context, *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	return nil, status.Error(codes.NotFound, "not found")
}
