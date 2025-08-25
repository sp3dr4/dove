package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sp3dr4/dove/config"
)

// PrometheusRegistry implements the Registry interface using Prometheus metrics
type PrometheusRegistry struct {
	registry *prometheus.Registry
	config   config.MetricsConfig

	// HTTP Metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	// Business Metrics
	urlsCreatedTotal    prometheus.Counter
	urlsRedirectedTotal prometheus.Counter
}

// NewPrometheusRegistry creates a new Prometheus metrics registry
func NewPrometheusRegistry(cfg config.MetricsConfig) (Registry, error) {
	registry := prometheus.NewRegistry()

	// Create HTTP metrics
	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{LabelMethod, LabelPath, LabelStatusCode},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{LabelMethod, LabelPath, LabelStatusCode},
	)

	httpRequestsInFlight := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	// Create business metrics
	urlsCreatedTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "urls_created_total",
			Help:      "Total number of URLs created",
		},
	)

	urlsRedirectedTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "urls_redirected_total",
			Help:      "Total number of URL redirects",
		},
	)

	// Register all metrics
	metricsCollectors := []prometheus.Collector{
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestsInFlight,
		urlsCreatedTotal,
		urlsRedirectedTotal,
	}

	for _, collector := range metricsCollectors {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}

	// Register Go runtime metrics if enabled
	if cfg.CollectRuntime {
		registry.MustRegister(collectors.NewGoCollector())
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	return &PrometheusRegistry{
		registry:             registry,
		config:               cfg,
		httpRequestsTotal:    httpRequestsTotal,
		httpRequestDuration:  httpRequestDuration,
		httpRequestsInFlight: httpRequestsInFlight,
		urlsCreatedTotal:     urlsCreatedTotal,
		urlsRedirectedTotal:  urlsRedirectedTotal,
	}, nil
}

// RecordHTTPRequest records an HTTP request with method, path, status code, and duration
func (p *PrometheusRegistry) RecordHTTPRequest(method, path, statusCode string, duration float64) {
	labels := prometheus.Labels{
		LabelMethod:     method,
		LabelPath:       path,
		LabelStatusCode: statusCode,
	}
	p.httpRequestsTotal.With(labels).Inc()
	p.httpRequestDuration.With(labels).Observe(duration)
}

// IncHTTPRequestsInFlight increments the in-flight HTTP requests counter
func (p *PrometheusRegistry) IncHTTPRequestsInFlight() {
	p.httpRequestsInFlight.Inc()
}

// DecHTTPRequestsInFlight decrements the in-flight HTTP requests counter
func (p *PrometheusRegistry) DecHTTPRequestsInFlight() {
	p.httpRequestsInFlight.Dec()
}

// IncURLsCreated increments the URLs created counter
func (p *PrometheusRegistry) IncURLsCreated() {
	p.urlsCreatedTotal.Inc()
}

// IncURLsRedirected increments the URLs redirected counter
func (p *PrometheusRegistry) IncURLsRedirected() {
	p.urlsRedirectedTotal.Inc()
}

// GetRegistry returns the underlying Prometheus registry
func (p *PrometheusRegistry) GetRegistry() *prometheus.Registry {
	return p.registry
}

// GetHandler returns an HTTP handler for the metrics endpoint
func (p *PrometheusRegistry) GetHandler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}
