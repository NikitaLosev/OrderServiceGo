package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"orderservice/pkg/api/orderpb"
	"orderservice/pkg/models"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type HTTPServer struct {
	gateway *runtime.ServeMux
}

var _ models.Order

// handleOrder proxies HTTP calls to gRPC gateway.
//
//	@Summary		Get order by UID
//	@Description	Returns order by uid from cache or DB
//	@Tags			orders
//	@Param			order_uid	path		string	true	"Order UID"
//	@Success		200			{object}	models.Order
//	@Failure		400			{string}	string
//	@Failure		404			{string}	string
//	@Router			/order/{order_uid} [get]
func (s *HTTPServer) handleOrder(w http.ResponseWriter, r *http.Request) {
	s.gateway.ServeHTTP(w, r)
}

func StartHTTPServer(ctx context.Context, addr string, grpcAddr string, logger *slog.Logger) error {
	gatewayMux := runtime.NewServeMux(
		runtime.WithErrorHandler(runtime.DefaultHTTPErrorHandler),
		runtime.WithMetadata(func(ctx context.Context, r *http.Request) metadata.MD {
			if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
				return metadata.Pairs("x-request-id", reqID)
			}
			return metadata.MD{}
		}),
	)

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
	if err := orderpb.RegisterOrderServiceHandlerFromEndpoint(ctx, gatewayMux, grpcAddr, dialOpts); err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	srv := &HTTPServer{gateway: gatewayMux}
	mux := http.NewServeMux()
	mux.Handle("/order/", http.HandlerFunc(srv.handleOrder))
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/order/") {
			srv.handleOrder(w, r)
			return
		}
		http.FileServer(http.Dir("static")).ServeHTTP(w, r)
	}))

	handler := chainMiddlewares(otelhttp.NewHandler(mux, "http.server"), logger)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
