package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"orderservice/internal/observability"
)

func TestNormalizePath(t *testing.T) {
	if got := normalizePath("/order/123"); got != "/order/{order_uid}" {
		t.Fatalf("normalize order path: %s", got)
	}
	if got := normalizePath("/swagger/index.html"); got != "/swagger" {
		t.Fatalf("normalize swagger path: %s", got)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	var captured string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = observability.RequestIDFromContext(r.Context())
	})

	requestIDMiddleware(logger)(handler).ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatalf("request id header not set")
	}
	if captured == "" {
		t.Fatalf("request id not in context")
	}
}
