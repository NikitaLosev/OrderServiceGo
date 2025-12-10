package server

import (
	"log/slog"
	"net/http"
	"time"

	"LZero/internal/observability"

	"github.com/google/uuid"
)

func requestIDMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = uuid.NewString()
			}
			ctx := observability.WithRequestID(r.Context(), reqID)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", reqID)
			next.ServeHTTP(w, r)
		})
	}
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)

			reqID := observability.RequestIDFromContext(r.Context())
			l := logger
			if reqID != "" {
				l = l.With("req_id", reqID)
			}
			l.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func chainMiddlewares(handler http.Handler, logger *slog.Logger) http.Handler {
	h := observeMetrics(handler)
	h = loggingMiddleware(logger)(h)
	h = requestIDMiddleware(logger)(h)
	return h
}
