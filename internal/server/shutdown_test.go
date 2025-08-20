package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"
)

func TestServerShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	if err := StartHTTPServer(ctx, addr, nil, logger); err != nil {
		t.Fatalf("server error: %v", err)
	}
}
