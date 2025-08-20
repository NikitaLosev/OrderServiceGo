package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"LZero/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler func(http.ResponseWriter, *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(err, service.ErrValidation):
			status = http.StatusBadRequest
		}
		http.Error(w, http.StatusText(status), status)
	}
}

func logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// StartHTTPServer runs HTTP server and shuts down on context cancel.
func StartHTTPServer(ctx context.Context, addr string, pool *pgxpool.Pool, logger *slog.Logger) error {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("static")))
	mux.Handle("/order/", orderHandler(pool))

	srv := &http.Server{Addr: addr, Handler: logging(logger, mux)}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func orderHandler(pool *pgxpool.Pool) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		uid := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/order/"))
		if uid == "" {
			return service.ErrValidation
		}
		if o, ok := service.OrderCache.Get(uid); ok {
			w.Header().Set("Content-Type", "application/json")
			return json.NewEncoder(w).Encode(o)
		}
		o, ok, err := service.GetOrderFromDB(pool, uid)
		if err != nil {
			return err
		}
		if !ok {
			return service.ErrNotFound
		}
		service.OrderCache.Set(uid, o)
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(o)
	}
}
