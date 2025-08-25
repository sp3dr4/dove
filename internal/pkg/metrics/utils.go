package metrics

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// GetRoutePath extracts the route pattern from the request context
// This helps group metrics by route pattern rather than specific values
func GetRoutePath(r *http.Request) string {
	// Try to get the route pattern from chi router context
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}

	// Fallback to request path, but normalize common patterns
	path := r.URL.Path
	return NormalizePath(path)
}

// NormalizePath normalizes URL paths to reduce cardinality in metrics
// This prevents metrics explosion from dynamic path segments
func NormalizePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	// Handle common API patterns
	switch {
	case path == "/health":
		return "/health"
	case path == "/ready":
		return "/ready"
	case path == "/metrics":
		return "/metrics"
	case path == "/shorten":
		return "/shorten"
	case strings.HasPrefix(path, "/swagger"):
		return "/swagger/*"
	case path == "/redoc":
		return "/redoc"
	default:
		// For short code redirects, normalize to pattern
		// Path like "/abc123" becomes "/{shortCode}"
		segments := strings.Split(strings.Trim(path, "/"), "/")
		if len(segments) == 1 && segments[0] != "" {
			// This is likely a short code redirect
			return "/{shortCode}"
		}
	}

	return path
}

// GetStatusCodeClass returns the HTTP status code class (2xx, 3xx, 4xx, 5xx)
// This can be useful for high-level metrics grouping
func GetStatusCodeClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// FormatStatusCode converts an integer status code to string
func FormatStatusCode(statusCode int) string {
	return strconv.Itoa(statusCode)
}

// SanitizeLabel sanitizes a string to be used as a Prometheus label value
// Removes or replaces characters that might cause issues
func SanitizeLabel(value string) string {
	// Replace common problematic characters
	value = strings.ReplaceAll(value, "\"", "")
	value = strings.ReplaceAll(value, "\\", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")

	// Limit length to prevent extremely long labels
	if len(value) > 100 {
		value = value[:100]
	}

	return value
}
