package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of incoming HTTP requests.",
	}, []string{"method", "path"})

	latencyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency distributions.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP 5xx responses.",
	}, []string{"method", "path"})
)

func init() {
	prometheus.MustRegister(requestCounter, latencyHistogram, errorCounter)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func observeMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)

		path := normalizePath(r.URL.Path)
		labels := prometheus.Labels{"method": r.Method, "path": path}
		requestCounter.With(labels).Inc()
		latencyHistogram.With(labels).Observe(time.Since(start).Seconds())
		if rec.status >= 500 {
			errorCounter.With(labels).Inc()
		}
	})
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "/order/") {
		return "/order/{order_uid}"
	}
	if strings.HasPrefix(path, "/swagger") {
		return "/swagger"
	}
	return path
}
