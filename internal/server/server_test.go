package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"LZero/internal/service"
)

func TestErrorMapping(t *testing.T) {
	rr := httptest.NewRecorder()
	h := handler(func(w http.ResponseWriter, r *http.Request) error { return service.ErrValidation })
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", rr.Code)
	}
}
