package metrics

import (
	"net/http"
	"time"
)

const (
	// MetricsPath is the default path for the metrics endpoint
	MetricsPath = "/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200 if WriteHeader is never called
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// PrometheusMiddleware creates HTTP middleware that records Prometheus metrics
func PrometheusMiddleware(registry Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics collection for the metrics endpoint itself to avoid recursion
			if r.URL.Path == MetricsPath {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Increment in-flight requests
			registry.IncHTTPRequestsInFlight()
			defer registry.DecHTTPRequestsInFlight()

			// Wrap the response writer to capture status code
			ww := newResponseWriter(w)

			// Process the request
			next.ServeHTTP(ww, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			method := r.Method
			path := GetRoutePath(r)
			statusCode := FormatStatusCode(ww.statusCode)

			registry.RecordHTTPRequest(method, path, statusCode, duration)
		})
	}
}

// MetricsMiddleware is an alias for PrometheusMiddleware for backward compatibility
var MetricsMiddleware = PrometheusMiddleware
