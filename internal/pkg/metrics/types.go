package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// Registry defines the interface for metrics collection
type Registry interface {
	// HTTP Metrics
	RecordHTTPRequest(method, path, statusCode string, duration float64)
	IncHTTPRequestsInFlight()
	DecHTTPRequestsInFlight()

	// Business Metrics
	IncURLsCreated()
	IncURLsRedirected()

	// Prometheus-specific methods
	GetRegistry() *prometheus.Registry
	GetHandler() http.Handler
}

// NoOpRegistry provides a no-op implementation for when metrics are disabled
type NoOpRegistry struct{}

func NewNoOpRegistry() Registry {
	return &NoOpRegistry{}
}

func (n *NoOpRegistry) RecordHTTPRequest(method, path, statusCode string, duration float64) {}
func (n *NoOpRegistry) IncHTTPRequestsInFlight()                                            {}
func (n *NoOpRegistry) DecHTTPRequestsInFlight()                                            {}
func (n *NoOpRegistry) IncURLsCreated()                                                     {}
func (n *NoOpRegistry) IncURLsRedirected()                                                  {}
func (n *NoOpRegistry) GetRegistry() *prometheus.Registry                                   { return nil }
func (n *NoOpRegistry) GetHandler() http.Handler                                            { return nil }

// MetricsLabels contains common label names used across metrics
type MetricsLabels struct {
	Method       string
	Path         string
	StatusCode   string
	Operation    string
	Status       string
	CacheStatus  string
	DatabaseType string
}

// Common label names as constants
const (
	LabelMethod       = "method"
	LabelPath         = "path"
	LabelStatusCode   = "status_code"
	LabelOperation    = "operation"
	LabelStatus       = "status"
	LabelCacheStatus  = "cache_status"
	LabelDatabaseType = "database_type"
)
