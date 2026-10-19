package metrics

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "HTTP requests.",
	}, []string{"method", "route", "status"})

	reqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "HTTP latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

func init() { prometheus.MustRegister(reqTotal, reqDuration) }

func Handler() http.Handler { return promhttp.Handler() }

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush keeps SSE and streaming responses working.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real writer (hijack, deadlines).
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// ChiMiddleware records count and latency per route template.
func ChiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		route := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
		}
		reqTotal.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
		reqDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

func Serve(addr, apiKey string) {
	if apiKey == "" {
		slog.Error("metrics server NOT started: METRICS_API_KEY is empty")
		return
	}
	want := sha256.Sum256([]byte(apiKey))

	authed := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := sha256.Sum256([]byte(r.Header.Get("X-Api-Key")))
			if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", authed(promhttp.Handler()))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("metrics server NOT started: cannot bind", "addr", addr, "err", err)
		return
	}
	slog.Info("metrics server started", "addr", addr)

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server stopped", "err", err)
		}
	}()
}
func RegisterDB(db *sql.DB, name string) {
	prometheus.MustRegister(collectors.NewDBStatsCollector(db, name))
}
