package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"LZero/internal/observability"
	"LZero/internal/service"
	"LZero/pkg/api/orderpb"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type orderGRPCServer struct {
	orderpb.UnimplementedOrderServiceServer
	svc    *service.Service
	logger *slog.Logger
	tracer trace.Tracer
}

func StartGRPCServer(ctx context.Context, addr string, svc *service.Service, logger *slog.Logger, tracer trace.Tracer) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			requestIDUnaryInterceptor(logger),
		),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	orderpb.RegisterOrderServiceServer(grpcServer, &orderGRPCServer{svc: svc, logger: logger, tracer: tracer})

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

func (s *orderGRPCServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	ctx, span := s.tracer.Start(ctx, "grpc.GetOrder")
	defer span.End()

	order, err := s.svc.GetOrder(ctx, req.GetOrderUid())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, service.ErrValidation):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &orderpb.GetOrderResponse{Order: modelToProto(order)}, nil
}

func requestIDUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		reqID := ""
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			reqID = vals[0]
		}
		if reqID == "" {
			reqID = observability.RequestIDFromContext(ctx)
		}
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx = observability.WithRequestID(ctx, reqID)
		if reqID != "" {
			logger = logger.With("req_id", reqID)
		}
		resp, err := handler(ctx, req)
		return resp, err
	}
}
