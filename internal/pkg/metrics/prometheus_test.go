package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sp3dr4/dove/config"
)

func TestNewPrometheusRegistry(t *testing.T) {
	tests := []struct {
		name   string
		config config.MetricsConfig
		want   bool // whether we expect success
	}{
		{
			name: "valid config",
			config: config.MetricsConfig{
				Enabled:         true,
				Path:            "/metrics",
				Namespace:       "dove",
				Subsystem:       "urlshortener",
				CollectRuntime:  true,
				CollectDatabase: true,
				CollectCache:    true,
			},
			want: true,
		},
		{
			name: "minimal config",
			config: config.MetricsConfig{
				Enabled:   true,
				Path:      "/metrics",
				Namespace: "test",
				Subsystem: "test",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := NewPrometheusRegistry(tt.config)

			if tt.want {
				require.NoError(t, err)
				assert.NotNil(t, registry)

				// Test that we can get the underlying registry
				promRegistry := registry.GetRegistry()
				assert.NotNil(t, promRegistry)

				// Test that we can get the handler
				handler := registry.GetHandler()
				assert.NotNil(t, handler)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestPrometheusRegistry_HTTPMetrics(t *testing.T) {
	config := config.MetricsConfig{
		Enabled:   true,
		Path:      "/metrics",
		Namespace: "test",
		Subsystem: "test",
	}

	registry, err := NewPrometheusRegistry(config)
	require.NoError(t, err)

	// Test HTTP request recording
	registry.RecordHTTPRequest("GET", "/test", "200", 0.1)
	registry.RecordHTTPRequest("POST", "/shorten", "201", 0.05)
	registry.RecordHTTPRequest("GET", "/abc123", "301", 0.02)

	// Test in-flight requests
	registry.IncHTTPRequestsInFlight()
	registry.IncHTTPRequestsInFlight()
	registry.DecHTTPRequestsInFlight()

	// We can't easily test the actual metric values without exposing them,
	// but we can verify the methods don't panic
	assert.NotPanics(t, func() {
		registry.RecordHTTPRequest("GET", "/test", "404", 0.01)
	})
}

func TestPrometheusRegistry_BusinessMetrics(t *testing.T) {
	config := config.MetricsConfig{
		Enabled:   true,
		Path:      "/metrics",
		Namespace: "test",
		Subsystem: "test",
	}

	registry, err := NewPrometheusRegistry(config)
	require.NoError(t, err)

	// Test business metrics
	assert.NotPanics(t, func() {
		registry.IncURLsCreated()
		registry.IncURLsCreated()
		registry.IncURLsRedirected()
	})
}

func TestNoOpRegistry(t *testing.T) {
	registry := NewNoOpRegistry()

	// All methods should be safe to call and not panic
	assert.NotPanics(t, func() {
		registry.RecordHTTPRequest("GET", "/test", "200", 0.1)
		registry.IncHTTPRequestsInFlight()
		registry.DecHTTPRequestsInFlight()
		registry.IncURLsCreated()
		registry.IncURLsRedirected()

		// These should return nil for NoOp
		assert.Nil(t, registry.GetRegistry())
		assert.Nil(t, registry.GetHandler())
	})
}
