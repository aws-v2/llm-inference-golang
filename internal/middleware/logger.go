package middleware

import (
	"log"
	"net"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger middleware
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default
		}

		// Get client IP
		ip := getClientIP(r)

		// Process request
		next.ServeHTTP(rw, r)

		// Log after request finishes
		duration := time.Since(start)

		log.Printf(
			"%s | %d | %s | %s %s | %s",
			ip,
			rw.statusCode,
			duration,
			r.Method,
			r.URL.Path,
			r.UserAgent(),
		)
	})
}

// Extract client IP safely
func getClientIP(r *http.Request) string {
	// Check proxy headers first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fallback to remote addr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}