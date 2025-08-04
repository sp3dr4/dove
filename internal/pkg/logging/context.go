package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

type contextKey string

const (
	loggerKey    contextKey = "logger"
	traceIDKey   contextKey = "trace_id"
	requestIDKey contextKey = "request_id"
)

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext extracts a logger from the context, falling back to default if not found
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts trace ID from context
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts request ID from context
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GenerateTraceID generates a new trace ID
func GenerateTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple counter-based approach if crypto/rand fails
		return "trace-" + hex.EncodeToString([]byte("fallback"))[:16]
	}
	return hex.EncodeToString(bytes)
}

// NewRequestLogger creates a logger with request context (request ID, trace ID)
func NewRequestLogger(ctx context.Context, baseLogger *slog.Logger) *slog.Logger {
	args := []any{}

	// Add request ID from context
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		args = append(args, "request_id", reqID)
	}

	// Add trace ID
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		args = append(args, "trace_id", traceID)
	}

	return baseLogger.With(args...)
}
