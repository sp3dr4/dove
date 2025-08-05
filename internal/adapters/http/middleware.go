package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/sp3dr4/dove/internal/pkg/logging"
)

// LoggingMiddleware creates HTTP middleware that injects request-scoped logger
// This adapter bridges chi-specific middleware with our generic logging package
func LoggingMiddleware(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()

			if reqID := middleware.GetReqID(ctx); reqID != "" {
				ctx = logging.WithRequestID(ctx, reqID)
			}

			traceID := r.Header.Get("X-Trace-Id")
			if traceID == "" {
				traceID = logging.GenerateTraceID()
			}
			ctx = logging.WithTraceID(ctx, traceID)

			w.Header().Set("X-Trace-Id", traceID)

			requestLogger := logging.NewRequestLogger(ctx, baseLogger)
			ctx = logging.WithLogger(ctx, requestLogger)

			requestLogger.Info("Request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)

			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r.WithContext(ctx))

			duration := time.Since(start)
			requestLogger.Info("Request completed",
				"status_code", ww.statusCode,
				"duration_ms", float64(duration.Nanoseconds())/1e6,
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
